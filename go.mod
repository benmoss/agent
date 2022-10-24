module github.com/buildkite/agent/v3

go 1.18

require (
	github.com/DataDog/datadog-go/v5 v5.1.1
	github.com/aws/aws-sdk-go v1.44.66
	github.com/buildkite/bintest/v3 v3.1.0
	github.com/buildkite/interpolate v0.0.0-20200526001904-07f35b4ae251
	github.com/buildkite/shellwords v0.0.0-20180315084142-c3f497d1e000
	github.com/buildkite/yaml v0.0.0-20210326113714-4a3f40911396
	github.com/creack/pty v1.1.18
	github.com/denisbrodbeck/machineid v1.0.0
	github.com/gofrs/flock v0.8.1
	github.com/google/go-querystring v1.0.0
	github.com/mattn/go-zglob v0.0.0-20180803001819-2ea3427bfa53
	github.com/mitchellh/go-homedir v1.1.0
	github.com/nightlyone/lockfile v0.0.0-20180618180623-0ad87eef1443
	github.com/oleiade/reflections v0.0.0-20160817071559-0e86b3c98b2f
	github.com/opentracing/opentracing-go v1.2.0
	github.com/pborman/uuid v0.0.0-20170112150404-1b00554d8222
	github.com/pkg/errors v0.9.1
	github.com/qri-io/jsonschema v0.0.0-20180607150648-d0d3b10ec792
	github.com/rjeczalik/interfaces v0.1.1
	github.com/sergi/go-diff v1.0.0 // indirect
	github.com/stretchr/testify v1.8.0
	github.com/urfave/cli v1.22.9
	golang.org/x/crypto v0.0.0-20220315160706-3147a52a75dd
	golang.org/x/oauth2 v0.0.0-20220622183110-fd043fe589d2
	golang.org/x/sys v0.0.0-20220722155257-8c9f86f7a55f
	google.golang.org/api v0.91.0
	gopkg.in/DataDog/dd-trace-go.v1 v1.40.1
)

require (
	cloud.google.com/go/compute v1.9.0
	github.com/boz/go-logutil v0.1.0
	github.com/boz/kail v0.15.0
	github.com/boz/kcache v0.4.1-0.20200916082658-69f5988adb8d
	github.com/buildkite/roko v1.0.0
	github.com/sirupsen/logrus v1.9.0
	go.opentelemetry.io/contrib/propagators/aws v1.8.0
	go.opentelemetry.io/contrib/propagators/b3 v1.8.0
	go.opentelemetry.io/contrib/propagators/jaeger v1.8.0
	go.opentelemetry.io/contrib/propagators/ot v1.8.0
	go.opentelemetry.io/otel v1.9.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.8.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.8.0
	go.opentelemetry.io/otel/sdk v1.8.0
	go.opentelemetry.io/otel/trace v1.9.0
	golang.org/x/exp v0.0.0-20220428152302-39d4317da171
	k8s.io/api v0.25.3
	k8s.io/apimachinery v0.25.3
)

require (
	github.com/PuerkitoBio/purell v1.1.1 // indirect
	github.com/PuerkitoBio/urlesc v0.0.0-20170810143723-de5bf2ad4578 // indirect
	github.com/alecthomas/template v0.0.0-20160405071501-a0175ee3bccc // indirect
	github.com/alecthomas/units v0.0.0-20151022065526-2efee857e7cf // indirect
	github.com/apex/log v1.1.0 // indirect
	github.com/blakesmith/ar v0.0.0-20150311145944-8bd4349a67f2 // indirect
	github.com/boz/go-lifecycle v0.1.0 // indirect
	github.com/campoy/unique v0.0.0-20180121183637-88950e537e7e // indirect
	github.com/client9/codegen v0.0.0-20180316044450-92480ce66a06 // indirect
	github.com/emicklei/go-restful/v3 v3.8.0 // indirect
	github.com/fatih/color v1.13.0 // indirect
	github.com/go-openapi/jsonpointer v0.19.5 // indirect
	github.com/go-openapi/jsonreference v0.19.5 // indirect
	github.com/go-openapi/swag v0.19.14 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/google/gnostic v0.5.7-v3refs // indirect
	github.com/google/go-cmp v0.5.8 // indirect
	github.com/google/go-github/v25 v25.0.1 // indirect
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/goreleaser/godownloader v0.0.0-20190525212044-f9a1d47be7eb // indirect
	github.com/goreleaser/goreleaser v0.107.0 // indirect
	github.com/goreleaser/nfpm v0.11.0 // indirect
	github.com/imdario/mergo v0.3.6 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/kamilsk/retry/v4 v4.0.0 // indirect
	github.com/mattn/go-colorable v0.1.11 // indirect
	github.com/mattn/go-isatty v0.0.14 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	golang.org/x/sync v0.0.0-20220601150217-0de741cfad7f // indirect
	gopkg.in/alecthomas/kingpin.v2 v2.2.6 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	k8s.io/klog/v2 v2.70.1 // indirect
	k8s.io/kube-openapi v0.0.0-20220803162953-67bda5d908f1 // indirect
	k8s.io/utils v0.0.0-20220728103510-ee6ede2d64ed // indirect
	sigs.k8s.io/json v0.0.0-20220713155537-f223a00ba0e2 // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.2.3 // indirect
	sigs.k8s.io/yaml v1.2.0 // indirect
)

require (
	github.com/DataDog/datadog-agent/pkg/obfuscate v0.0.0-20211129110424-6491aa3bf583 // indirect
	github.com/DataDog/datadog-go v4.8.2+incompatible // indirect
	github.com/DataDog/sketches-go v1.2.1 // indirect
	github.com/Microsoft/go-winio v0.5.1 // indirect
	github.com/cenkalti/backoff/v4 v4.1.3 // indirect
	github.com/cespare/xxhash/v2 v2.1.2 // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.0-20190314233015-f79a8a8ca69d // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dgraph-io/ristretto v0.1.0 // indirect
	github.com/dustin/go-humanize v1.0.0 // indirect
	github.com/go-logr/logr v1.2.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/golang/glog v1.0.0 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/google/uuid v1.3.0 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.1.0 // indirect
	github.com/googleapis/gax-go/v2 v2.4.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.7.0 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/petermattis/goid v0.0.0-20180202154549-b0b1615b78e5 // indirect
	github.com/philhofer/fwd v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/qri-io/jsonpointer v0.0.0-20180309164927-168dd9e45cf2 // indirect
	github.com/russross/blackfriday/v2 v2.0.1 // indirect
	github.com/sanity-io/litter v1.5.5
	github.com/sasha-s/go-deadlock v0.0.0-20180226215254-237a9547c8a5 // indirect
	github.com/shurcooL/sanitized_anchor_name v1.0.0 // indirect
	github.com/tinylib/msgp v1.1.2 // indirect
	go.opencensus.io v0.23.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/internal/retry v1.8.0 // indirect
	go.opentelemetry.io/proto/otlp v0.18.0 // indirect
	go.uber.org/atomic v1.7.0 // indirect
	go.uber.org/multierr v1.8.0 // indirect
	golang.org/x/net v0.0.0-20220722155237-a158d28d115b // indirect
	golang.org/x/term v0.0.0-20210927222741-03fcf44c2211 // indirect
	golang.org/x/text v0.3.7 // indirect
	golang.org/x/time v0.0.0-20220210224613-90d013bbcef8 // indirect
	golang.org/x/tools v0.1.10 // indirect
	golang.org/x/xerrors v0.0.0-20220609144429-65e65417b02f // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/genproto v0.0.0-20220804142021-4e6b2dfa6612 // indirect
	google.golang.org/grpc v1.48.0 // indirect
	google.golang.org/protobuf v1.28.1 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	k8s.io/client-go v0.25.3
)

replace github.com/boz/kail => github.com/mnaser/kail v0.0.0-20220816210502-1cd6d8f901c3

replace github.com/boz/kcache => github.com/mnaser/kcache v0.0.0-20220816205242-e805863dfd0a
