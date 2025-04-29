package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"os/user"
	"regexp"
	"runtime"
	"strings"
	"time"
)

func main() {
	ctx := &XrayContext{}
	app := InitApp("Setup mkxray, please wait...")
	defer app.RestoreConsole()
	app.Jobs = []*Job{
		CheckIfProperSystem(),
		CheckIfRoot(),
		DownloadXray(),
		InstallXray(),
		CheckXray(),
		PickDestination(ctx, app),
		GenerateXrayContext(ctx),
		WriteXrayConfig(ctx),
		RestartXray(),
	}

	RefreshLines(app)
	RenderUI(app)
	for jobIdx := range app.Jobs {
		job := app.Jobs[jobIdx]
		err := RunJob(job, app)

		if err != nil {
			println(ErrorMsg(app, err.Error()))
			os.Exit(1)
		}
	}
	ClearUI(app)
	RefreshLines(app)
	AppendEndMessage(app, ctx)
	RenderUI(app)
}

func CheckIfProperSystem() *Job {
	return NewJob("Check system", func() error {
		sys := runtime.GOOS
		if sys != "linux" {
			return fmt.Errorf("found %s system; not linux", sys)
		}
		return nil
	})
}

func CheckIfRoot() *Job {
	return NewJob("Check if root", func() error {
		currentUser, err := user.Current()
		if err != nil {
			return fmt.Errorf("unable to get current user")
		}
		if currentUser.Username != "root" {
			return fmt.Errorf("not root user")
		}
		return nil
	})
}

func DownloadXray() *Job {
	return NewJob("Download xray installer", func() error {
		resp, err := http.Get(XRAY_INSTALL_URL)
		if err != nil {
			return fmt.Errorf("unable to download xray installer: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("failed to download xray installer: %s", resp.Status)
		}
		bytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("unable to read xray installer: %v", err)
		}
		installerPath := os.TempDir() + "/install-xray.sh"
		err = os.WriteFile(installerPath, bytes, 0777)
		if err != nil {
			return fmt.Errorf("unable to save xray installer: %v", err)
		}
		return nil
	})
}

func InstallXray() *Job {
	return NewJob("Install xray", func() error {
		installerPath := os.TempDir() + "/install-xray.sh"
		_, err := exec.Command(installerPath).Output()
		if err != nil {
			return fmt.Errorf("unable to run xray installer: %v", err)
		}
		return nil
	})
}

func CheckXray() *Job {
	return NewJob("Check xray installation", func() error {
		out, err := exec.Command(
			"xray", "--version", ">", "/dev/null", "2", ">", "&1",
			"&&", "echo", "0",
			"||", "echo", "1",
		).Output()
		if err != nil {
			return fmt.Errorf("unable to verify xray installation: %v", err)
		}
		if string(out[:]) == "1" {
			return fmt.Errorf("xray wasn't installed successfully")
		}
		return nil
	})
}

func PickDestination(ctx *XrayContext, app *App) *Job {
	knownDestinations := []string{
		"www.samsung.com:443",
		"www.asus.com:443",
		"www.microsoft.com:443",
	}
	j := NewJob("Pick website to mimick", nil)
	j.Execute = func() error {
		for i, dest := range knownDestinations {
			WriteJobOutput(fmt.Sprintf("%d. %s\n", i+1, dest), j, app)
		}
		WriteJobOutput("Choose one of sites to mimick:\n", j, app)
		time.Sleep(3 * time.Second)
		choice := knownDestinations[0]
		WriteJobOutput("You chose: "+choice+"\n", j, app)
		ctx.Dest = choice
		time.Sleep(3 * time.Second)
		ClearJobOutput(j, app)
		return nil
	}
	return j
}

func GenerateXrayContext(ctx *XrayContext) *Job {
	return NewJob("Generate xray context", func() error {
		serverName := strings.SplitN(ctx.Dest, ":", 1)[0]
		if serverName == "" {
			return fmt.Errorf("unable to get server name from destination")
		}
		key, pubKey, err := NewXrayKeys()
		if err != nil {
			return fmt.Errorf("unable to generate xray keys: %v", err)
		}
		shortID, err := NewShortID()
		if err != nil {
			return fmt.Errorf("unable to generate short ID: %v", err)
		}
		clientID, err := NewXrayUuid()
		if err != nil {
			return fmt.Errorf("unable to generate xray UUID: %v", err)
		}
		externalID, err := GetExternalIP()
		if err != nil {
			return fmt.Errorf("unable to get external IP: %v", err)
		}

		ctx.ServerName = serverName
		ctx.PrivateKey = key
		ctx.PublicKey = pubKey
		ctx.ClientID = clientID
		ctx.ShortID = shortID
		ctx.ExternalIP = externalID
		ctx.VlessLink = GenerateVlessLink(ctx, "mkxray", "xtls-rprx-vision", "raw", "reality", "edge")
		return nil
	})
}

func WriteXrayConfig(ctx *XrayContext) *Job {
	return NewJob("Write Xray config", func() error {
		cfg := strings.NewReplacer(
			"$dest$", ctx.Dest,
			"$clientID$", ctx.ClientID,
			"$serverName$", ctx.ServerName,
			"$privateKey$", ctx.PrivateKey,
			"$shortID$", ctx.ShortID,
		).Replace(XRAY_CONFIG_TEMPLATE)
		err := os.WriteFile(XRAY_CONFIG_PATH, []byte(cfg), 0444)
		if err != nil {
			return fmt.Errorf("unable to write xray config file: %v", err)
		}
		return nil
	})
}

func RestartXray() *Job {
	return NewJob("Restart xray", func() error {
		_, err := exec.Command("systemctl", "restart", "xray").Output()
		if err != nil {
			return fmt.Errorf("unable to restart xray: %v", err)
		}
		time.Sleep(2 * time.Second)

		out, err := exec.Command(
			"journalctl",
			"-u", "xray",
			"-n", "1",
			"--no-pager",
		).Output()
		if err != nil {
			return fmt.Errorf("unable to get xray logs: %v", err)
		}
		re := regexp.MustCompile("core: Xray .* started")
		if re.Find(out) == nil {
			return fmt.Errorf("something went wrong during xray restarting: %s", out)
		}
		return nil
	})
}

// internal

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

type XrayContext struct {
	Dest       string
	ServerName string
	PrivateKey string
	PublicKey  string
	ClientID   string
	ShortID    string
	ExternalIP string
	VlessLink  string
}

func NewXrayUuid() (string, error) {
	out, err := exec.Command("xray", "uuid").Output()
	if err != nil {
		return "", err
	}
	return string(out[0 : len(out)-1]), nil
}

func NewXrayKeys() (string, string, error) {
	out, err := exec.Command("xray", "x25519").Output()
	if err != nil {
		return "", "", err
	}
	re := regexp.MustCompile("Private key: (.+)\nPublic key: (.+)\n")
	groups := re.FindStringSubmatch(string(out[:]))
	private_key := groups[1]
	public_key := groups[2]
	return private_key, public_key, nil
}

func NewShortID() (string, error) {
	out, err := exec.Command("openssl", "rand", "-hex", "8").Output()
	if err != nil {
		return "", err
	}
	return string(out[0 : len(out)-1]), nil
}

func GetExternalIP() (string, error) {
	out, err := exec.Command("dig", "+short", "myip.opendns.com", "@resolver1.opendns.com").Output()
	if err != nil {
		return "", err
	}
	return string(out[0 : len(out)-1]), nil
}

func GenerateVlessLink(ctx *XrayContext, name, flow, typ, security, fp string) string {
	link := fmt.Sprintf(
		"vless://%s@%s:443?flow=%s&type=%s&security=%s&fp=%s&sni=%s&pbk=%s&sid=%s#%s",
		ctx.ClientID,
		ctx.ExternalIP,
		flow,
		typ,
		security,
		fp,
		ctx.ServerName,
		ctx.PublicKey,
		ctx.ShortID,
		name,
	)
	return link
}
