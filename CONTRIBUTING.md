# Contributing

## Build Tool

```sh
go install github.com/magefile/mage@latest
go install mvdan.cc/gofumpt@latest
```

## Format

```sh
gofumpt -l -w .
```

## Postman

- host: `http://localhost:23456`
- container: `https://localhost:443`

```js
let jsonData = pm.response.json();
pm.collectionVariables.set("apiKey", jsonData.token);
```

- Login body

```json
{
  "password": "root",
  "username": "root"
}
```

## Docs

- Install redocly/cli

```sh
npm i -g @redocly/cli@latest
```

## Tools

### Conventional Commit

- install git cz tool global

```sh
npm install -g commitizen
npm install -g cz-conventional-changelog
npm install -g conventional-changelog-cli
echo '{ "path": "cz-conventional-changelog" }' > ~/.czrc
```

### Pre-commit

- install pre-commit in any way you like

```sh
pre-commit autoupdate
pre-commit install
pre-commit run --all-files
```

## Modify CHANGELOG

- git-chglog

```sh
brew tap git-chglog/git-chglog
brew install git-chglog
```

```sh
VERSION=1.0.0
git tag -a v$VERSION -m $VERSION
git push -u origin --tags
git push -u origin --all
```

## Find ignored files

```sh
find . -type f  | git check-ignore --stdin
```
