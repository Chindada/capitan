package usecases

import (
	"context"
	"fmt"
	"net/mail"
	"time"

	"github.com/chindada/capitan/internal/config"
	"github.com/chindada/capitan/internal/usecases/modules/encrypt"
	"github.com/chindada/capitan/internal/usecases/repo"
	"github.com/chindada/leopard/pkg/eventbus"
	"github.com/chindada/leopard/pkg/log"
	"github.com/chindada/panther/golang/pb"
	"github.com/chindada/panther/pkg/launcher"
	"github.com/gin-gonic/gin"
	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
	"github.com/ricochet2200/go-disk-usage/du"
	"github.com/sethvargo/go-password/password"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/protobuf/types/known/timestamppb"
)

//go:generate mockgen -source=usecase_system.go -destination=./mocks/mocks_usecase_system_test.go -package=mocks

const (
	defaultUserRoot   = "root"
	defaultMailDomain = "example.com"
)

const (
	totpOrgName = "AutumnXP"
	totpOrgMail = "info@autumnxp.com"
)

type System interface {
	GetDiskUsage() *du.DiskUsage
	GetLaunchTime() time.Time

	Login(ctx *gin.Context, loginReq *pb.LoginRequest) (*pb.User, error)

	CreateTotp(username string) (*otp.Key, error)
	ValidateTotp(key, code string) bool
	AddTotpByUser(ctx context.Context, username string, totp *pb.Totp) error

	GetLastJWT(ctx context.Context) (string, error)
	InsertJWT(ctx context.Context, jwt string) error

	GetUser(ctx context.Context, username string) (*pb.User, error)
	CreateUser(ctx context.Context, t *pb.User) error
	GetAllUser(ctx context.Context) (*pb.UserList, error)
	UpdateUser(ctx context.Context, t *pb.User) error
	DeleteUser(ctx context.Context, username string) error
	ChangePassword(ctx context.Context, username, oldPassword, newPassword string) error
}

type systemUseCase struct {
	systemRepo repo.SystemRepo
	userRepo   repo.UserRepo

	logger *log.Log
	bus    *eventbus.Bus

	launchTime time.Time
}

func NewSystem() System {
	logger := log.Get()
	cfg := config.Get()
	pg := cfg.GetPostgresPool()
	uc := &systemUseCase{
		launchTime: time.Now(),
		systemRepo: repo.NewSystemRepo(pg),
		userRepo:   repo.NewUserRepo(pg),
		logger:     logger,
		bus:        eventbus.Get(),
	}
	uc.initUsers()
	return uc
}

func (uc *systemUseCase) initUsers() {
	all, err := uc.userRepo.SelectAllUser(context.Background())
	if err != nil {
		uc.logger.Fatal(err)
		return
	}
	if len(all.GetList()) == 0 {
		res, gErr := password.Generate(32, 4, 0, false, false)
		if gErr != nil {
			uc.logger.Fatal(gErr)
		}
		root := &pb.User{
			Basic: &pb.BasicUser{
				Email:    fmt.Sprintf("%s@%s", defaultUserRoot, defaultMailDomain),
				Username: defaultUserRoot,
				Password: res,
				Role:     pb.UserRole_ROOT,
			},
		}
		uc.logger.Infof("No user found, creating %s user: %s",
			defaultUserRoot,
			root.GetBasic().GetPassword())
		if err = uc.CreateUser(context.Background(), root); err != nil {
			uc.logger.Fatal(err)
		}
	}
}

func (uc *systemUseCase) CreateUser(ctx context.Context, t *pb.User) error {
	_, err := mail.ParseAddress(t.GetBasic().GetEmail())
	if err != nil {
		return ErrEmailFormatInvalid
	}

	if t.GetBasic().GetRole() != pb.UserRole_ROOT &&
		t.GetBasic().GetRole() != pb.UserRole_ADMIN &&
		t.GetBasic().GetRole() != pb.UserRole_USER {
		return ErrRoleInvalid
	}

	all, err := uc.userRepo.SelectAllUser(ctx)
	if err != nil {
		return err
	}

	for _, user := range all.GetList() {
		if user.GetBasic().GetUsername() == t.GetBasic().GetUsername() {
			return ErrUsernameAlreadyExists
		}
		if user.GetBasic().GetEmail() == t.GetBasic().GetEmail() {
			return ErrEmailAlreadyExists
		}
	}

	t.Basic.Password, err = encrypt.Encrypt(t.GetBasic().GetPassword())
	if err != nil {
		return err
	}
	if err = uc.userRepo.InsertUser(ctx, t); err != nil {
		return err
	}
	return nil
}

func (uc *systemUseCase) GetLastJWT(ctx context.Context) (string, error) {
	s, err := uc.systemRepo.SelectSetting(ctx, pb.SettingKey_SETTING_JWT)
	if err != nil {
		return "", err
	}
	return s.GetJwt().GetSecret(), nil
}

func (uc *systemUseCase) InsertJWT(ctx context.Context, jwt string) error {
	return uc.systemRepo.InsertSetting(ctx, &pb.SystemSetting{
		Key: pb.SettingKey_SETTING_JWT,
		Value: &pb.SystemSetting_Jwt{
			Jwt: &pb.SettingJWT{
				Secret: jwt,
			},
		},
	})
}

func (uc *systemUseCase) Login(ctx *gin.Context, loginReq *pb.LoginRequest) (*pb.User, error) {
	var err error
	var code pb.LoginRespCode
	user, err := uc.userRepo.SelectUserByUsername(ctx, loginReq.GetUsername())
	if err != nil {
		code = pb.LoginRespCode_DB_ERROR
		return nil, err
	}
	defer func() {
		event := &pb.LoginEvent{
			User:      user,
			Ip:        ctx.ClientIP(),
			RespCode:  code,
			CreatedAt: timestamppb.Now(),
		}
		_ = uc.systemRepo.InsertLoginEvent(ctx, []*pb.LoginEvent{event})
	}()
	if user.GetBasic().GetUsername() == "" {
		code = pb.LoginRespCode_USER_NOT_FOUND
		return nil, ErrUserNotFound
	}
	err = bcrypt.CompareHashAndPassword([]byte(user.GetBasic().GetPassword()), []byte(loginReq.GetPassword()))
	if err != nil {
		code = pb.LoginRespCode_PASSWORD_INCORRECT
		return nil, ErrPasswordNotMatch
	}
	if !user.GetEnableTotp() || user.GetTotpId() == 0 {
		return user, nil
	}
	if loginReq.GetMfaCode() == "" {
		return nil, ErrMfaCodeRequired
	}
	totpKey, err := uc.userRepo.SelectTotpByID(ctx, user.GetTotpId())
	if err != nil {
		code = pb.LoginRespCode_DB_ERROR
		return nil, err
	}
	if !uc.ValidateTotp(totpKey.GetSecret(), loginReq.GetMfaCode()) {
		code = pb.LoginRespCode_MFA_FAILED
		return nil, ErrMfaCodeNotMatch
	}
	return user, nil
}

func (uc *systemUseCase) GetUser(ctx context.Context, username string) (*pb.User, error) {
	user, err := uc.userRepo.SelectUserByUsername(ctx, username)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (uc *systemUseCase) GetAllUser(ctx context.Context) (*pb.UserList, error) {
	return uc.userRepo.SelectAllUser(ctx)
}

func (uc *systemUseCase) UpdateUser(ctx context.Context, t *pb.User) error {
	return uc.userRepo.UpdateUser(ctx, t)
}

func (uc *systemUseCase) DeleteUser(ctx context.Context, username string) error {
	return uc.userRepo.DeleteUser(ctx, username)
}

func (uc *systemUseCase) ChangePassword(ctx context.Context, username, oldPassword, newPassword string) error {
	user, err := uc.userRepo.SelectUserByUsername(ctx, username)
	if err != nil {
		return err
	}
	if user.GetBasic().GetUsername() == "" {
		return ErrUserNotFound
	}
	err = bcrypt.CompareHashAndPassword([]byte(user.GetBasic().GetPassword()), []byte(oldPassword))
	if err != nil {
		return ErrPasswordNotMatch
	}
	newPass, err := encrypt.Encrypt(newPassword)
	if err != nil {
		return err
	}
	user.Basic.Password = newPass
	if err = uc.userRepo.UpdateUserPassword(ctx, user); err != nil {
		return err
	}
	return nil
}

func (uc *systemUseCase) CreateTotp(username string) (*otp.Key, error) {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      totpOrgName,
		AccountName: username,
	})
	if err != nil {
		return nil, err
	}
	return key, nil
}

func (uc *systemUseCase) AddTotpByUser(ctx context.Context, username string, totp *pb.Totp) error {
	user, err := uc.userRepo.SelectUserByUsername(ctx, username)
	if err != nil {
		return err
	}
	return uc.userRepo.ActivateUserTotp(ctx, user, totp)
}

func (uc *systemUseCase) ValidateTotp(key, code string) bool {
	return totp.Validate(code, key)
}

func (uc *systemUseCase) GetDiskUsage() *du.DiskUsage {
	return du.NewDiskUsage(launcher.Get().GetDataPath())
}

func (uc *systemUseCase) GetLaunchTime() time.Time {
	return uc.launchTime
}
