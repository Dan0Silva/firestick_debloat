# 🔥 FireStick Debloat (`firestick_debloat`)

![Go Version](https://img.shields.io/badge/Go-1.22%2B-00ADD8?style=flat-square&logo=go)
![Platform](https://img.shields.io/badge/Platform-Linux%20%7C%20macOS%20%7C%20Windows-informational?style=flat-square)
![License](https://img.shields.io/badge/License-MIT-green?style=flat-square)

A fast, zero-dependency, interactive CLI/TUI tool written in **Pure Go** to debloat, strip telemetry, and optimize your **Amazon Fire TV Stick** over your local network.

No need to install `adb` or Android SDK Platform-Tools on your computer — just clone, build, and run!

---

## ✨ Features

- **🚀 Zero External Dependencies (Pure Go ADB)**: Connects directly to your Fire Stick over TCP/IP (`:5555`) using a native Go ADB implementation.
- **🖥️ Interactive Terminal UI (TUI)**: Built with [Charmbracelet Bubble Tea](https://github.com/charmbracelet/bubbletea). Easily toggle individual tweaks or choose **Select All** before clicking **Execute**.
- **🧹 Amazon Bloatware Removal**: Disables/removes pre-installed Amazon bloatware (Amazon Music, Kids/FreeTime, Silk Browser, Luna, GameCircle, Audible, Live TV channels bloat, etc.) while **safely preserving Amazon Prime Video and Amazon Photos**.
- **🛡️ Telemetry & Ad Tracking Blocker**: Disables Amazon telemetry, crash reporters, usage metrics, marketing services, and limits advertising ID tracking.
- **⚡ System Optimizations**:
  - Reduces UI animation scales (`window`, `transition`, and `animator` scales to `0.5x`) for a noticeably snappier navigation experience.
  - Limits unnecessary background processes and services.
  - Trims and clears accumulated application caches (`pm trim-caches`).
- **🔒 Safe & Non-Destructive**: Operates at the user level (`--user 0`) without requiring root access. Core FireOS system stability, **Prime Video**, and **Amazon Photos** are protected by a built-in whitelist.

---

## 🛠️ Prerequisites (On Your Fire Stick)

Before running the tool, enable **ADB Debugging** on your Fire TV Stick and find its local IP address:

1. On your Fire Stick home screen, go to **Settings** ⚙️ > **My Fire TV**.
2. If **Developer Options** is not visible:
   - Select **About**.
   - Highlight your **Fire TV Stick** device name and press the **Select (Center)** button on your remote **7 times** rapidly until you see *"No need, you are already a developer"*.
   - Press the **Back** button to return to **My Fire TV**.
3. Open **Developer Options** and turn **ON** **ADB Debugging**.
4. Go back to **My Fire TV** > **About** > **Network** and note down your Fire Stick's **IP Address** (e.g., `192.168.1.100`).

---

## 🚀 Quick Start (Clone, Build & Run)

Make sure you have [Go 1.22+](https://go.dev/dl/) installed on your machine.

### 1. Clone the repository
```bash
git clone https://github.com/v0rtx/firestick_debloat.git
cd firestick_debloat
```

### 2. Build the binary
```bash
go build -o firestick_debloat ./cmd/firestick_debloat
# Or using Make:
make build
```

### 3. Run the tool with your Fire Stick IP
```bash
./firestick_debloat 192.168.1.100

# Optional: Start directly in Restore Mode to re-enable packages or default settings
./firestick_debloat --restore 192.168.1.100

# Optional: Explicitly repair/re-enable Prime Video, Amazon Photos & all shared dependencies
./firestick_debloat --fix-prime 192.168.1.100
```
*(Note: On your very first connection, look at your TV screen, check **"Always allow from this computer"**, and click **OK** on the USB/ADB Debugging prompt).*

---

## 🎮 How to Use the Interactive Menu

Once connected to your Fire Stick, the interactive menu will appear:

```text
┌──────────────────────────────────────────────────────────────────────────┐
│ 🔥 FireStick Debloat — Connected to 192.168.1.100:5555                   │
├──────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│   [ ] Select All Actions                                                 │
│                                                                          │
│ ── Amazon Bloatware (Preserves Prime Video & Photos) ─────────────────── │
│ > [x] Remove Amazon Music, Audible & Kindle Bloat                        │
│   [x] Remove Amazon Kids (FreeTime) & Tahiti                             │
│   [x] Remove Amazon Silk Browser                                         │
│   [x] Remove Amazon Luna, GameCircle & Appstore Promos                   │
│   [x] Remove Fire TV Live Channels, Freevee & Alexa Shopping             │
│                                                                          │
│ ── Privacy, Telemetry & Ads ──────────────────────────────────────────── │
│   [x] Disable Amazon Metrics, Telemetry & Crash Reporters                │
│   [x] Disable Ad Tracking & Personalized Marketing                       │
│                                                                          │
│ ── System Optimizations ──────────────────────────────────────────────── │
│   [x] Speed Up UI Animations (Set scales to 0.5x)                        │
│   [x] Limit Background Services & Trim App Caches                        │
│                                                                          │
│   [ ▶ EXECUTE SELECTED ACTIONS ]                                         │
│                                                                          │
└──────────────────────────────────────────────────────────────────────────┘
  ↑/↓: Navigate • Space: Toggle • a: Select All • Enter: Execute • q: Quit
```

### Keyboard Controls
| Key | Action |
| :--- | :--- |
| `↑` / `↓` or `k` / `j` | Move cursor up / down |
| `Space` | Toggle (`[x]` / `[ ]`) the highlighted option |
| `a` | Toggle **Select All** options at once |
| `r` | Switch between **🔥 Debloat Mode** and **♻️ Restore Mode** |
| `Enter` | Trigger **Execute** when on the Execute button (or toggle item) |
| `q` / `Ctrl+C` | Quit without applying changes |

---

## 📦 What Gets Modified?

### 1. Amazon Pre-Installed Apps (Bloatware)
> **🛡️ Protected Apps:** **Amazon Prime Video** (`com.amazon.avod`, `com.amazon.firebat`, `com.amazon.hedwig`, and its required `.api`/`tcomm` libraries) and **Amazon Photos** (`com.amazon.bueller.photos`) are **strictly excluded** from removal so your core media experience works out of the box.

| Target Group | Packages Disabled / Uninstalled (`--user 0`) |
| :--- | :--- |
| **Music & Books** | `com.amazon.mp3`, `com.audible.application.kindle`, `com.amazon.kindle`, `com.amazon.unifiedsharegoodreads` |
| **Kids / FreeTime** | `com.amazon.tahoe`, `com.amazon.recess` |
| **Web Browser** | `com.amazon.cloud9` |
| **Gaming & Promos** | `com.amazon.ags.app`, `com.amazon.gloria.smarthome` |
| **Shopping Bloat** | `com.amazon.alexashopping`, `com.amazon.shoptv.client` |

### 2. Telemetry, Ads & Data Collection
| Target Group | Actions Performed |
| :--- | :--- |
| **Metrics & Telemetry** | Disables `com.amazon.device.metrics`, `com.amazon.client.metrics` *(preserves `.api` libraries needed by Prime Video)* |
| **Crash & Log Collectors** | Disables `com.amazon.device.crashmanager`, `com.amazon.csapp`, `com.amazon.dp.logger`, `com.amazon.H2ClientSvc` |
| **Ad Personalization** | Disables `com.amazon.advertisingidsettings`, `com.amazon.kso.blackbird` and sets `settings put secure limit_ad_tracking 1` |

### 3. System Optimizations
| Optimization | Command / Mechanism |
| :--- | :--- |
| **Faster UI Animations** | Sets `window_animation_scale`, `transition_animation_scale`, and `animator_duration_scale` to `0.5` |
| **Background Optimization** | Disables background sync bloatware & limits idle background services |
| **Cache Cleanup** | Executes `pm trim-caches 999999999999999999` to free up internal storage |

---

## ⚠️ Disclaimer

This tool is provided **as-is** without warranty of any kind. While all commands run in non-root user space (`--user 0`) and have been curated to keep FireOS stable, firmware updates may vary across Fire TV Stick generations. Always review the options you select before executing. If everything breaks, a standard Factory Reset in FireOS settings will restore all default packages.

---

## 📄 License

Distributed under the **MIT License**. Feel free to fork, contribute, and open issues or pull requests!
