package version

import (
	"embed"
	"encoding/json"

	"github.com/Masterminds/semver/v3"
	"github.com/chindada/panther/golang/pb"
)

var (
	//go:embed core.json
	coreFile embed.FS

	//go:embed fronted.json
	frontedFile embed.FS
)

func unknownVersion() *pb.SystemBuild {
	return &pb.SystemBuild{
		Version: "unknown",
		Commit:  "unknown",
	}
}

func unmarshal(data []byte) *pb.SystemBuild {
	v := pb.SystemBuild{}
	err := json.Unmarshal(data, &v)
	if err != nil {
		panic(err)
	}
	_, err = semver.NewVersion(v.GetVersion())
	if err != nil {
		return unknownVersion()
	}
	return &v
}

func GetCore() *pb.SystemBuild {
	data, err := coreFile.ReadFile("core.json")
	if err != nil {
		panic(err)
	}
	return unmarshal(data)
}

func GetWeb() *pb.SystemBuild {
	data, err := frontedFile.ReadFile("fronted.json")
	if err != nil {
		panic(err)
	}
	return unmarshal(data)
}
