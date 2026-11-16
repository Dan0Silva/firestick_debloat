package debloat

// DefaultCatalog returns the curated list of categories and debloat/optimization actions
// for Amazon Fire TV Stick devices.
// Note: Amazon Prime Video (com.amazon.avod, com.amazon.firebat, com.amazon.hedwig, and its
// required shared libraries) and Amazon Photos (com.amazon.bueller.photos) are strictly
// excluded and protected by the whitelist.
func DefaultCatalog() []Category {
	return []Category{
		{
			ID:          CategoryBloatware,
			Title:       "Amazon Bloatware (Preserves Prime Video & Photos)",
			Description: "Disable pre-installed Amazon apps while keeping Prime Video and Photos intact",
			Actions: []Action{
				{
					ID:          "bloat_music_books",
					Category:    CategoryBloatware,
					Title:       "Amazon Music, Audible & Kindle Bloat",
					Description: "Targets Amazon Music, Audible, Kindle, and Goodreads integration",
					Steps: []CommandStep{
						{Type: StepDisablePackage, Target: "com.amazon.mp3", Description: "Disable Amazon Music", RestoreDescription: "Re-enable Amazon Music"},
						{Type: StepDisablePackage, Target: "com.audible.application.kindle", Description: "Disable Audible", RestoreDescription: "Re-enable Audible"},
						{Type: StepDisablePackage, Target: "com.amazon.kindle", Description: "Disable Amazon Kindle", RestoreDescription: "Re-enable Amazon Kindle"},
						{Type: StepDisablePackage, Target: "com.amazon.unifiedsharegoodreads", Description: "Disable Goodreads Share", RestoreDescription: "Re-enable Goodreads Share"},
					},
				},
				{
					ID:          "bloat_kids",
					Category:    CategoryBloatware,
					Title:       "Amazon Kids (FreeTime) & Tahiti",
					Description: "Targets Amazon Kids/FreeTime parental bloatware services",
					Steps: []CommandStep{
						{Type: StepDisablePackage, Target: "com.amazon.tahoe", Description: "Disable Amazon Kids (FreeTime)", RestoreDescription: "Re-enable Amazon Kids (FreeTime)"},
						{Type: StepDisablePackage, Target: "com.amazon.recess", Description: "Disable Amazon Recess Kids Service", RestoreDescription: "Re-enable Amazon Recess Kids Service"},
					},
				},
				{
					ID:          "bloat_browser",
					Category:    CategoryBloatware,
					Title:       "Amazon Silk Browser App",
					Description: "Targets the standalone Amazon Silk Browser app while keeping system WebView providers intact",
					Steps: []CommandStep{
						{Type: StepDisablePackage, Target: "com.amazon.cloud9", Description: "Disable Amazon Silk Browser", RestoreDescription: "Re-enable Amazon Silk Browser"},
					},
				},
				{
					ID:          "bloat_gaming",
					Category:    CategoryBloatware,
					Title:       "Amazon Luna, GameCircle & SmartHome Promos",
					Description: "Targets GameCircle and SmartHome Gloria promotional clients",
					Steps: []CommandStep{
						{Type: StepDisablePackage, Target: "com.amazon.ags.app", Description: "Disable Amazon GameCircle", RestoreDescription: "Re-enable Amazon GameCircle"},
						{Type: StepDisablePackage, Target: "com.amazon.gloria.smarthome", Description: "Disable Gloria SmartHome Bloat", RestoreDescription: "Re-enable Gloria SmartHome"},
					},
				},
				{
					ID:          "bloat_shopping",
					Category:    CategoryBloatware,
					Title:       "Alexa Shopping & ShopTV Bloat",
					Description: "Targets Alexa Shopping and ShopTV apps without touching Prime Video or Live TV",
					Steps: []CommandStep{
						{Type: StepDisablePackage, Target: "com.amazon.alexashopping", Description: "Disable Alexa Shopping", RestoreDescription: "Re-enable Alexa Shopping"},
						{Type: StepDisablePackage, Target: "com.amazon.shoptv.client", Description: "Disable Amazon ShopTV Client", RestoreDescription: "Re-enable Amazon ShopTV Client"},
					},
				},
			},
		},
		{
			ID:          CategoryPrivacy,
			Title:       "Privacy, Telemetry & Ads",
			Description: "Block Amazon metrics, crash reporting, and personalized ad tracking (preserves Prime Video APIs)",
			Actions: []Action{
				{
					ID:          "priv_telemetry",
					Category:    CategoryPrivacy,
					Title:       "Amazon Background Metrics Collectors",
					Description: "Targets standalone background metric collectors (preserves .api libraries needed by Prime Video)",
					Steps: []CommandStep{
						{Type: StepDisablePackage, Target: "com.amazon.device.metrics", Description: "Disable Device Metrics Collector", RestoreDescription: "Re-enable Device Metrics Collector"},
						{Type: StepDisablePackage, Target: "com.amazon.client.metrics", Description: "Disable Client Metrics Collector", RestoreDescription: "Re-enable Client Metrics Collector"},
					},
				},
				{
					ID:          "priv_crash_logs",
					Category:    CategoryPrivacy,
					Title:       "Crash Reporters & Diagnostic Loggers",
					Description: "Targets background crash dumps and diagnostic log uploaders",
					Steps: []CommandStep{
						{Type: StepDisablePackage, Target: "com.amazon.device.crashmanager", Description: "Disable Crash Manager", RestoreDescription: "Re-enable Crash Manager"},
						{Type: StepDisablePackage, Target: "com.amazon.csapp", Description: "Disable Customer Service Diagnostic App", RestoreDescription: "Re-enable Customer Service Diagnostic App"},
						{Type: StepDisablePackage, Target: "com.amazon.dp.logger", Description: "Disable DP Logger", RestoreDescription: "Re-enable DP Logger"},
						{Type: StepDisablePackage, Target: "com.amazon.H2ClientSvc", Description: "Disable H2 Client Diagnostic Service", RestoreDescription: "Re-enable H2 Client Diagnostic Service"},
					},
				},
				{
					ID:          "priv_ads",
					Category:    CategoryPrivacy,
					Title:       "Ad Tracking & Promotional Banners",
					Description: "Targets Advertising ID settings, Blackbird promos, and enables limit_ad_tracking",
					Steps: []CommandStep{
						{Type: StepDisablePackage, Target: "com.amazon.advertisingidsettings", Description: "Disable Advertising ID Settings", RestoreDescription: "Re-enable Advertising ID Settings"},
						{Type: StepDisablePackage, Target: "com.amazon.kso.blackbird", Description: "Disable Blackbird Promotional Service", RestoreDescription: "Re-enable Blackbird Promotional Service"},
						{
							Type:               StepShellCommand,
							Target:             "settings put secure limit_ad_tracking 1",
							Description:        "Enable Limit Ad Tracking in Secure Settings",
							RestoreTarget:      "settings put secure limit_ad_tracking 0",
							RestoreDescription: "Restore Default Ad Tracking in Secure Settings",
						},
					},
				},
			},
		},
		{
			ID:          CategoryOptimizations,
			Title:       "System Optimizations",
			Description: "Speed up FireOS UI animations, optimize lifecycle settings, and free up cache",
			Actions: []Action{
				{
					ID:          "opt_animations",
					Category:    CategoryOptimizations,
					Title:       "Speed Up UI Animations (0.5x / Restore 1.0x)",
					Description: "Sets window, transition, and animator duration scales to 0.5x (or restores 1.0x)",
					Steps: []CommandStep{
						{
							Type:               StepShellCommand,
							Target:             "settings put global window_animation_scale 0.5",
							Description:        "Set Window Animation Scale to 0.5x",
							RestoreTarget:      "settings put global window_animation_scale 1.0",
							RestoreDescription: "Restore Window Animation Scale to 1.0x",
						},
						{
							Type:               StepShellCommand,
							Target:             "settings put global transition_animation_scale 0.5",
							Description:        "Set Transition Animation Scale to 0.5x",
							RestoreTarget:      "settings put global transition_animation_scale 1.0",
							RestoreDescription: "Restore Transition Animation Scale to 1.0x",
						},
						{
							Type:               StepShellCommand,
							Target:             "settings put global animator_duration_scale 0.5",
							Description:        "Set Animator Duration Scale to 0.5x",
							RestoreTarget:      "settings put global animator_duration_scale 1.0",
							RestoreDescription: "Restore Animator Duration Scale to 1.0x",
						},
					},
				},
				{
					ID:          "opt_background",
					Category:    CategoryOptimizations,
					Title:       "Optimize Background Activity Lifecycle",
					Description: "Ensures standard Android activity management without killing Prime Video services",
					Steps: []CommandStep{
						{
							Type:               StepShellCommand,
							Target:             "settings put global always_finish_activities 0",
							Description:        "Ensure standard activity lifecycle management",
							RestoreTarget:      "settings put global always_finish_activities 0",
							RestoreDescription: "Restore standard activity lifecycle management",
						},
					},
				},
				{
					ID:          "opt_cache",
					Category:    CategoryOptimizations,
					Title:       "Trim & Clear Application Caches",
					Description: "Runs pm trim-caches to reclaim internal storage space",
					Steps: []CommandStep{
						{
							Type:               StepShellCommand,
							Target:             "pm trim-caches 999999999999999999",
							Description:        "Trim all application caches",
							RestoreTarget:      "",
							RestoreDescription: "Cache trim does not require restoration",
						},
					},
				},
			},
		},
	}
}

// AllActions returns a flattened slice of all Actions across all Categories in DefaultCatalog.
func AllActions() []Action {
	var all []Action
	for _, cat := range DefaultCatalog() {
		all = append(all, cat.Actions...)
	}
	return all
}
