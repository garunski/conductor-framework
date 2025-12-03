package api

import (
	"fmt"
	"strconv"
	"strings"

	"k8s.io/apimachinery/pkg/version"

	"github.com/garunski/conductor-framework/pkg/framework/manifest"
)

func (h *Handler) checkKubernetesVersion(appReq manifest.ApplicationRequirement, discovery interface{ ServerVersion() (*version.Info, error) }, versionInfo *version.Info) *ClusterRequirement {
	if versionInfo == nil {
		var err error
		versionInfo, err = discovery.ServerVersion()
		if err != nil {
			return &ClusterRequirement{
				Name:        appReq.Name,
				Description: appReq.Description,
				Status:      "fail",
				Message:     fmt.Sprintf("Unable to check version: %v", err),
				Required:    appReq.Required,
			}
		}
	}

	// Get minimum version from config, default to 1.24
	minVersion := "1.24"
	if minVer, ok := appReq.CheckConfig["minimumVersion"].(string); ok {
		minVersion = minVer
	}

	// Parse version
	major, _ := strconv.Atoi(strings.TrimPrefix(versionInfo.Major, "v"))
	minor, _ := strconv.Atoi(strings.Split(versionInfo.Minor, "+")[0])
	versionStr := fmt.Sprintf("%s.%s", versionInfo.Major, versionInfo.Minor)

	// Parse minimum version
	minParts := strings.Split(minVersion, ".")
	minMajor, _ := strconv.Atoi(minParts[0])
	minMinor := 0
	if len(minParts) > 1 {
		minMinor, _ = strconv.Atoi(minParts[1])
	}

	status := "pass"
	message := fmt.Sprintf("Version %s meets requirement (>= %s)", versionStr, minVersion)
	if major < minMajor || (major == minMajor && minor < minMinor) {
		status = "fail"
		message = fmt.Sprintf("Version %s is below minimum required (%s)", versionStr, minVersion)
	}

	return &ClusterRequirement{
		Name:        appReq.Name,
		Description: appReq.Description,
		Status:      status,
		Message:     message,
		Required:    appReq.Required,
	}
}

