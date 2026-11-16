package debloat

import (
	"fmt"
	"strings"
)

// ProtectedPackages is the immutable whitelist of packages that must NEVER be disabled or removed.
// Explicitly protects Amazon Prime Video (both avod and firebat plus shared APIs),
// Amazon Photos, and core FireOS system components.
var ProtectedPackages = map[string]string{
	// Core Media Apps explicitly preserved by project design (Prime Video & Photos)
	"com.amazon.avod":           "Amazon Prime Video Player/Service (Protected)",
	"com.amazon.firebat":        "Amazon Prime Video Fire TV Client/Bridge (Protected)",
	"com.amazon.hedwig":         "Amazon Prime Video Channels & Playback Helper (Protected)",
	"com.amazon.bueller.photos": "Amazon Photos & Screensaver (Protected)",

	// Shared libraries & services required by Prime Video to launch and authenticate without crashing
	"com.amazon.client.metrics.api": "Amazon Metrics API Shared Library (Required by Prime Video)",
	"com.amazon.metrics.api":        "Amazon System Metrics API Library (Required by Prime Video)",
	"com.amazon.tcomm":              "Amazon TComm Device Communication (Required by Prime Video)",
	"com.amazon.tcomm.client":       "Amazon TComm Client Service (Required by Prime Video)",
	"com.amazon.aca":                "Amazon Content Adaptation Service (Required by Prime Video)",
	"com.amazon.predelivery":        "Amazon Media Pre-Delivery Service (Required by Prime Video)",
	"com.amazon.SYNC.service":       "Amazon Account Sync Service (Required by Prime Video)",

	// Core FireOS UI, Settings, Networking & Account Authentication (required for Prime Video/Photos login)
	"com.amazon.tv.launcher":      "FireOS Home Launcher",
	"com.amazon.firehomestarter":  "FireOS Home Starter",
	"com.amazon.tv.settings.v2":   "FireOS Settings UI",
	"com.amazon.device.settings":  "FireOS Device Settings Provider",
	"com.amazon.imp":              "Amazon Identity Mobile Platform (Account Auth)",
	"com.amazon.dcp":              "Amazon Device Client Platform (Core Auth/Sync)",
	"com.amazon.ssm":              "System Status Monitor",
	"com.amazon.tv.nimh":          "Network & Wi-Fi Management",
	"com.amazon.device.bluetooth": "Bluetooth & Remote Control Service",
	"android":                     "Android System Core",
	"com.android.systemui":        "Android System UI",
	"com.android.settings":        "Android Base Settings",
}

// PrimeVideoRepairPackages lists all packages that must be actively enabled on the Fire Stick
// to guarantee that Amazon Prime Video and Amazon Photos work properly (including healing
// devices where an older debloat list disabled com.amazon.firebat or shared APIs).
var PrimeVideoRepairPackages = []string{
	"com.amazon.avod",
	"com.amazon.firebat",
	"com.amazon.hedwig",
	"com.amazon.bueller.photos",
	"com.amazon.client.metrics.api",
	"com.amazon.metrics.api",
	"com.amazon.tcomm",
	"com.amazon.tcomm.client",
	"com.amazon.aca",
	"com.amazon.predelivery",
	"com.amazon.SYNC.service",
	"com.amazon.tv.livetv",
	"com.amazon.imdb.tv.android.app",
	"com.amazon.cloud9.contentservice",
	"com.amazon.cloud9.systembrowserprovider",
}

// IsWhitelisted returns true if the given package name is in the ProtectedPackages whitelist.
func IsWhitelisted(pkg string) bool {
	clean := strings.TrimSpace(pkg)
	_, exists := ProtectedPackages[clean]
	return exists
}

// WhitelistReason returns the protection description if the package is whitelisted.
func WhitelistReason(pkg string) (string, bool) {
	clean := strings.TrimSpace(pkg)
	reason, exists := ProtectedPackages[clean]
	return reason, exists
}

// ValidateCatalog inspects all actions and steps in the catalog to guarantee that no
// protected package is ever targeted for removal or disabling.
func ValidateCatalog(categories []Category) error {
	for _, cat := range categories {
		for _, act := range cat.Actions {
			if len(act.Steps) == 0 {
				return fmt.Errorf("action %q (%s) has no command steps", act.ID, act.Title)
			}
			for _, step := range act.Steps {
				if step.Type == StepDisablePackage {
					if reason, protected := WhitelistReason(step.Target); protected {
						return fmt.Errorf(
							"whitelist violation in action %q: package %q is protected (%s)",
							act.ID, step.Target, reason,
						)
					}
				}
				if step.Type == StepShellCommand {
					for pkg, reason := range ProtectedPackages {
						if pkg == "android" {
							continue
						}
						if strings.Contains(step.Target, pkg) {
							return fmt.Errorf(
								"whitelist violation in shell command of action %q: references protected package %q (%s)",
								act.ID, pkg, reason,
							)
						}
					}
				}
			}
		}
	}
	return nil
}
