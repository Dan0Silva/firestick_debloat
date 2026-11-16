package debloat

import "testing"

func TestDefaultCatalogRespectsWhitelist(t *testing.T) {
	catalog := DefaultCatalog()
	if len(catalog) != 3 {
		t.Fatalf("expected 3 categories in DefaultCatalog, got %d", len(catalog))
	}

	if err := ValidateCatalog(catalog); err != nil {
		t.Fatalf("DefaultCatalog failed whitelist validation: %v", err)
	}
}

func TestPrimeVideoAndPhotosExplicitlyWhitelisted(t *testing.T) {
	mustProtect := []string{
		"com.amazon.avod",               // Amazon Prime Video Player
		"com.amazon.firebat",            // Amazon Prime Video Fire TV App/Bridge
		"com.amazon.hedwig",             // Amazon Prime Video Channels Helper
		"com.amazon.client.metrics.api", // Shared library required by Prime Video
		"com.amazon.metrics.api",        // Shared library required by Prime Video
		"com.amazon.tcomm",              // TComm required by Prime Video
		"com.amazon.bueller.photos",     // Amazon Photos
		"com.amazon.tv.launcher",        // FireOS Launcher
	}

	for _, pkg := range mustProtect {
		if !IsWhitelisted(pkg) {
			t.Errorf("expected package %q to be protected by whitelist, but IsWhitelisted returned false", pkg)
		}
	}
}

func TestValidateCatalogDetectsViolation(t *testing.T) {
	badCatalog := []Category{
		{
			ID:    CategoryBloatware,
			Title: "Bad Category",
			Actions: []Action{
				{
					ID:    "bad_action",
					Title: "Accidental Prime Video Removal",
					Steps: []CommandStep{
						{Type: StepDisablePackage, Target: "com.amazon.avod", Description: "Should be blocked"},
					},
				},
			},
		},
	}

	if err := ValidateCatalog(badCatalog); err == nil {
		t.Fatalf("expected ValidateCatalog to reject catalog containing com.amazon.avod, but got nil error")
	}
}
