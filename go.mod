module github.com/flanksource/commons

go 1.12

require (
	github.com/flanksource/yaml v0.0.0-20200322131016-b7b2608b8702
	github.com/hairyhenderson/gomplate v3.5.0+incompatible
	github.com/hashicorp/go-getter v1.3.1-0.20190906090232-a0f878cb75da
	github.com/hashicorp/vault/api v1.0.4
	github.com/kr/pretty v0.2.0
	github.com/onsi/gomega v1.9.0
	github.com/pkg/errors v0.9.1
	github.com/sirupsen/logrus v1.4.2
	github.com/smartystreets/assertions v1.0.0 // indirect
	github.com/vbauerster/mpb/v5 v5.0.3
	golang.org/x/crypto v0.0.0-20200317142112-1b76d66859c6
	gopkg.in/hairyhenderson/yaml.v2 v2.2.2 // indirect
)

replace gopkg.in/hairyhenderson/yaml.v2 => github.com/maxaudron/yaml v0.0.0-20190411130442-27c13492fe3c
