package version

var tag = "dev"
var buildTime = "?"
var commit = "?"
var version = "dev"

type Version struct {
	Tag       string `json:"tag"`
	BuildTime string `json:"build_time"` // Time when application was build.
	Commit    string `json:"commit"`     // Git commit hash.
	Version   string `json:"version"`    // Version of the application.

}

func Get() *Version {
	return &Version{
		Version:   version,
		BuildTime: buildTime,
		Commit:    commit,
		Tag:       tag,
	}
}
