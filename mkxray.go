package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"os/user"
	"regexp"
	"strings"
	"time"

	"github.com/fatih/color"
)

func main() {
	ExitIfNotRoot()
	UpdateSystem()
	InstallXray()
	PrintInfo("Generating items for xray\n")
	ctx := NewXrayContext("www.samsung.com:443", "www.samsung.com")
	cfg := MakeXrayConfig(ctx)
	WriteConfigFile(cfg)
	RestartXray()
	PrintInfo("Xray service is ready for usage\n")
}

func ExitIfNotRoot() {
	currentUser, err := user.Current()
	HandleError(err, "Unable to get current user")
	if currentUser.Username != "root" {
		PrintErr("`mkxray` is supposed to be executed with elevated privileges. Please restart it using `sudo mkxray`\n")
		os.Exit(1)
	}
}

func UpdateSystem() {
	PrintInfo("Updating system...\n")
	err := RunCmd("apt-get", "update", "-y")
	HandleError(err, "Unable to update system")

	PrintInfo("Upgrading system...\n")
	err = RunCmd("apt-get", "upgrade", "-y")
	HandleError(err, "Unable to upgrade system")
}

func InstallXray() {
	PrintInfo("Downloading Xray installer...\n")
	resp, err := http.Get(XRAY_INSTALL_URL)
	HandleError(err, "Unable to download xray installer")
	bytes, err := io.ReadAll(resp.Body)
	HandleError(err, "Unable to read xray installer")

	installerPath := os.TempDir() + "/install-xray.sh"
	err = os.WriteFile(installerPath, bytes, 0777)
	HandleError(err, "Unable to write xray installer")

	PrintInfo("Running installer...\n")
	err = RunCmd("./install-xray.sh")
	HandleError(err, "Unable to run xray installer")

	PrintInfo("Verifying installation...\n")
	out, err := exec.Command("xray", "--version", ">", "/dev/null", "2", ">", "&1", "&&", "echo", "0", "||", "echo", "1").Output()
	HandleError(err, "Unable to verify xray installation")
	if string(out[:]) == "1" {
		fmt.Println()
		PrintErr("Xray wasn't installed successfully; exiting...\n")
		os.Exit(1)
	}
}

func NewXrayContext(dest, serverName string) *XrayContext {
	key, pubKey := NewXrayKeys()
	ctx := XrayContext{
		dest:       dest,
		serverName: serverName,
		privateKey: key,
		public_key: pubKey,
		clientID:   NewXrayUuid(),
		shortID:    NewShortID(),
	}
	return &ctx
}

func MakeXrayConfig(ctx *XrayContext) string {
	r := strings.NewReplacer(
		"$dest$", ctx.dest,
		"$clientID$", ctx.clientID,
		"$serverName$", ctx.serverName,
		"$privateKey$", ctx.privateKey,
		"$shortID$", ctx.shortID,
	)
	return r.Replace(XRAY_CONFIG_TEMPLATE)
}

func WriteConfigFile(cfg string) {
	PrintInfo(fmt.Sprintf("Updating config file at `%s`\n", XRAY_CONFIG_PATH))
	err := os.WriteFile(XRAY_CONFIG_PATH, []byte(cfg), 0444)
	HandleError(err, "Unable to write xray config file")
}

func RestartXray() {
	PrintInfo("Restarting xray...\n")
	RunCmd("systemctl", "restart", "xray")
	fmt.Println()
	time.Sleep(2 * time.Second)

	PrintInfo("Verifying xray is running fine...\n")
	out, err := exec.Command(
		"journalctl",
		"-u", "xray",
		"-n", "1",
		"--no-pager",
	).Output()
	HandleError(err, "Unable to get xray logs")
	re := regexp.MustCompile("core: Xray .* started")
	if re.Find(out) == nil {
		PrintErr("Something went wrong during xray restarting:\n")
		out, _ = exec.Command(
			"journalctl",
			"-u", "xray",
			"-n", "6",
			"--no-pager",
		).Output()
		fmt.Println(string(out))
		PrintErr("Run `journalctl -u xray` to more details\n")
		os.Exit(1)
	}
}

const (
	XRAY_INSTALL_URL     = "https://raw.githubusercontent.com/XTLS/Xray-install/046d9aa2432b3a6241d73c3684ef4e512974b594/install-release.sh"
	XRAY_CONFIG_PATH     = "/usr/local/etc/xray/config.json"
	XRAY_CONFIG_TEMPLATE = `{
  "log": {
    "loglevel": "info"
  },
  "routing": {
    "rules": [],
    "domainStrategy": "AsIs"
  },
  "inbounds": [
    {
      "port": 23,
      "tag": "ss",
      "protocol": "shadowsocks",
      "settings": {
        "method": "2022-blake3-aes-128-gcm",
        "password": "aaaaaaaaaaaaaaaabbbbbbbbbbbbbbbb",
        "network": "tcp,udp"
      }
    },
    {
      "port": 443,
      "protocol": "vless",
      "tag": "vless_tls",
      "settings": {
        "clients": [
          {
            "id": "$clientID$",
            "email": "user1@myserver",
            "flow": "xtls-rprx-vision"
          }
        ],
        "decryption": "none"
      },
      "streamSettings": {
        "network": "tcp",
        "security": "reality",
        "realitySettings": {
          "show": false,
          "dest": "$dest$",
          "xver": 0,
          "serverNames": [
            "$serverName$"
          ],
          "privateKey": "$privateKey$",
          "minClientVer": "",
          "maxClientVer": "",
          "maxTimeDiff": 0,
          "shortIds": [
            "$shortID$"
          ]
        }
      },
      "sniffing": {
        "enabled": true,
        "destOverride": [
          "http",
          "tls"
        ]
      }
    }
  ],
  "outbounds": [
    {
      "protocol": "freedom",
      "tag": "direct"
    },
    {
      "protocol": "blackhole",
      "tag": "block"
    }
  ]
}`
)

// internal

type XrayContext struct {
	dest       string
	serverName string
	privateKey string
	public_key string
	clientID   string
	shortID    string
}

func HandleError(err error, msg string) {
	if err != nil {
		PrintErr(fmt.Sprintf("%s: %s\n", msg, err))
		os.Exit(1)
	}
}

func PrintLvl(lvl, msg string) {
	fmt.Printf("[%s] %s", lvl, msg)
}

func PrintErr(msg string) {
	red := color.New(color.FgRed).SprintFunc()
	PrintLvl(red(" ERR"), msg)
}

func PrintInfo(msg string) {
	hiCyan := color.New(color.FgHiCyan).SprintFunc()
	PrintLvl(hiCyan("INFO"), msg)
}

func RunCmd(cmdName string, args ...string) error {
	cyan := color.New(color.FgCyan).SprintFunc()
	cmdString := cmdName + " " + strings.Join(args, " ")
	PrintLvl(cyan(" CMD"), cmdString)

	cmd := exec.Command(cmdName, args...)
	cmd.Stdout = os.Stdout
	err := cmd.Run()
	return err
}

func NewXrayUuid() string {
	out, err := exec.Command("xray", "uuid").Output()
	HandleError(err, "Unable to generate UUID")
	return string(out[0 : len(out)-1])
}

func NewXrayKeys() (string, string) {
	out, err := exec.Command("xray", "x25519").Output()
	HandleError(err, "Unable to generate xray keys")
	re := regexp.MustCompile("Private key: (.+)\nPublic key: (.+)\n")
	groups := re.FindStringSubmatch(string(out[:]))
	private_key := groups[1]
	public_key := groups[2]
	return private_key, public_key
}

func NewShortID() string {
	out, err := exec.Command("openssl", "rand", "-hex", "8").Output()
	HandleError(err, "Unable to generate short ID")
	return string(out[0 : len(out)-1])
}
