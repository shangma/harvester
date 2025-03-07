package settings

import (
	"encoding/json"
	"regexp"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
)

var (
	releasePattern = regexp.MustCompile("^v[0-9]")
	settings       = map[string]Setting{}
	provider       Provider
	InjectDefaults string

	APIUIVersion                 = NewSetting("api-ui-version", "1.1.9") // Please update the HARVESTER_API_UI_VERSION in package/Dockerfile when updating the version here.
	ServerVersion                = NewSetting("server-version", "dev")
	UIIndex                      = NewSetting("ui-index", DefaultDashboardUIURL)
	UIPath                       = NewSetting("ui-path", "/usr/share/harvester/harvester")
	UISource                     = NewSetting("ui-source", "auto") // Options are 'auto', 'external' or 'bundled'
	VolumeSnapshotClass          = NewSetting("volume-snapshot-class", "longhorn")
	BackupTargetSet              = NewSetting(BackupTargetSettingName, InitBackupTargetToString())
	UpgradableVersions           = NewSetting("upgradable-versions", "")
	UpgradeCheckerEnabled        = NewSetting("upgrade-checker-enabled", "true")
	UpgradeCheckerURL            = NewSetting("upgrade-checker-url", "https://harvester-upgrade-responder.rancher.io/v1/checkupgrade")
	LogLevel                     = NewSetting("log-level", "info") // options are info, debug and trace
	SupportBundleImage           = NewSetting("support-bundle-image", "rancher/support-bundle-kit:v0.0.3")
	SupportBundleImagePullPolicy = NewSetting("support-bundle-image-pull-policy", "IfNotPresent")
	DefaultStorageClass          = NewSetting("default-storage-class", "longhorn")
)

const (
	BackupTargetSettingName = "backup-target"
	DefaultDashboardUIURL   = "https://releases.rancher.com/harvester-ui/dashboard/latest/index.html"
)

func init() {
	if InjectDefaults == "" {
		return
	}
	defaults := map[string]string{}
	if err := json.Unmarshal([]byte(InjectDefaults), &defaults); err != nil {
		return
	}
	for name, defaultValue := range defaults {
		value, ok := settings[name]
		if !ok {
			continue
		}
		value.Default = defaultValue
		settings[name] = value
	}
}

type Provider interface {
	Get(name string) string
	Set(name, value string) error
	SetIfUnset(name, value string) error
	SetAll(settings map[string]Setting) error
}

type Setting struct {
	Name     string
	Default  string
	ReadOnly bool
}

func (s Setting) SetIfUnset(value string) error {
	if provider == nil {
		return s.Set(value)
	}
	return provider.SetIfUnset(s.Name, value)
}

func (s Setting) Set(value string) error {
	if provider == nil {
		s, ok := settings[s.Name]
		if ok {
			s.Default = value
			settings[s.Name] = s
		}
	} else {
		return provider.Set(s.Name, value)
	}
	return nil
}

func (s Setting) Get() string {
	if provider == nil {
		s := settings[s.Name]
		return s.Default
	}
	return provider.Get(s.Name)
}

func (s Setting) GetInt() int {
	v := s.Get()
	i, err := strconv.Atoi(v)
	if err == nil {
		return i
	}
	logrus.Errorf("failed to parse setting %s=%s as int: %v", s.Name, v, err)
	i, err = strconv.Atoi(s.Default)
	if err != nil {
		return 0
	}
	return i
}

func SetProvider(p Provider) error {
	if err := p.SetAll(settings); err != nil {
		return err
	}
	provider = p
	return nil
}

func NewSetting(name, def string) Setting {
	s := Setting{
		Name:    name,
		Default: def,
	}
	settings[s.Name] = s
	return s
}

func GetEnvKey(key string) string {
	return "HARVESTER_" + strings.ToUpper(strings.Replace(key, "-", "_", -1))
}

func IsRelease() bool {
	return !strings.Contains(ServerVersion.Get(), "head") && releasePattern.MatchString(ServerVersion.Get())
}

type TargetType string

const (
	S3BackupType  TargetType = "s3"
	NFSBackupType TargetType = "nfs"
)

type BackupTarget struct {
	Type               TargetType `json:"type"`
	Endpoint           string     `json:"endpoint"`
	AccessKeyID        string     `json:"accessKeyId"`
	SecretAccessKey    string     `json:"secretAccessKey"`
	BucketName         string     `json:"bucketName"`
	BucketRegion       string     `json:"bucketRegion"`
	Cert               string     `json:"cert"`
	VirtualHostedStyle bool       `json:"virtualHostedStyle"`
}

func InitBackupTargetToString() string {
	target := &BackupTarget{}
	targetStr, err := json.Marshal(target)
	if err != nil {
		logrus.Errorf("failed to init string backupTarget, error: %s", err.Error())
	}
	return string(targetStr)
}
