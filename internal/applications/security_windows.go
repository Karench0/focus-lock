//go:build windows

package applications

import (
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/yusufpapurcu/wmi"
)

var securityOnce sync.Once
var securityPaths []string

func securityDirectories() []string {
	securityOnce.Do(func() {
		// Security Center exposes registered third-party antivirus paths.
		// If WMI is unavailable, keep conservative vendor-directory exemptions.
		var products []struct {
			PathToSignedProductExe   string
			PathToSignedReportingExe string
		}
		if wmi.QueryNamespace("SELECT pathToSignedProductExe,pathToSignedReportingExe FROM AntiVirusProduct", &products, `root\SecurityCenter2`) == nil {
			for _, p := range products {
				for _, raw := range []string{p.PathToSignedProductExe, p.PathToSignedReportingExe} {
					i := strings.Index(strings.ToLower(raw), ".exe")
					if i < 0 {
						continue
					}
					exe := strings.Trim(raw[:i+4], ` "`)
					if !filepath.IsAbs(exe) {
						continue
					}
					dir := filepath.Dir(exe)
					if dir != filepath.VolumeName(dir)+`\` {
						securityPaths = append(securityPaths, strings.ToLower(dir)+`\`)
					}
				}
			}
		}
		vendors := []string{"Windows Defender", "Microsoft Defender", "Kaspersky Lab", "Kaspersky", "ESET", "AVAST Software", "AVG", "Bitdefender", "Norton", "NortonLifeLock", "Symantec", "McAfee", "Malwarebytes", "Trend Micro", "Sophos", "CrowdStrike", "SentinelOne", "Sentinel Agent", "F-Secure", "Panda Security", "Webroot", "COMODO", "DrWeb", "Doctor Web"}
		if base := os.Getenv("ProgramData"); base != "" {
			for _, vendor := range []string{"Windows Defender", "Microsoft Defender"} {
				securityPaths = append(securityPaths, strings.ToLower(filepath.Join(base, "Microsoft", vendor))+`\`)
			}
		}
		for _, base := range []string{os.Getenv("ProgramFiles"), os.Getenv("ProgramFiles(x86)")} {
			if base == "" {
				continue
			}
			for _, vendor := range vendors {
				securityPaths = append(securityPaths, strings.ToLower(filepath.Join(base, vendor))+`\`)
			}
		}
	})
	return securityPaths
}
