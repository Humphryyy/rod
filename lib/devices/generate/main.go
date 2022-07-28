package main

import (
	"fmt"
	"io/ioutil"
	"strings"

	http "github.com/Carcraftz/fhttp"

	"github.com/Humphryyy/rod/lib/utils"
	"github.com/ysmood/gson"
)

func main() {
	devices := getDeviceList()

	code := ``
	for _, d := range devices.Arr() {
		d = d.Get("device")
		name := d.Get("title").String()

		code += utils.S(`

			// {{.name}} device
			{{.name}} = Device{
				Title:        "{{.title}}",
				Capabilities: {{.capabilities}},
				UserAgent:    "{{.userAgent}}",
				AcceptLanguage: "en",
				Screen: Screen{
					DevicePixelRatio: {{.devicePixelRatio}},
					Horizontal: ScreenSize{
						Width: {{.horizontalWidth}},
						Height: {{.horizontalHeight}},
					},
					Vertical: ScreenSize{
						Width: {{.verticalWidth}},
						Height: {{.verticalHeight}},
					},
				},
			}`,
			"name", normalizeName(name),
			"title", name,
			"capabilities", toGoArr(d.Get("capabilities")),
			"userAgent", getUserAgent(d),
			"devicePixelRatio", d.Get("screen.device-pixel-ratio").Int(),
			"horizontalWidth", d.Get("screen.horizontal.width").Int(),
			"horizontalHeight", d.Get("screen.horizontal.height").Int(),
			"verticalWidth", d.Get("screen.vertical.width").Int(),
			"verticalHeight", d.Get("screen.vertical.height").Int(),
		)
	}

	code = utils.S(`// generated by "lib/devices/generate"

		package devices

		import (
			"github.com/Humphryyy/rod/lib/devices"
		)

		var (
			{{.code}}
		)
	`, "code", code)

	path := "./lib/devices/list.go"
	utils.E(utils.OutputFile(path, code))

	utils.Exec("gofmt -s -w", path)
	utils.Exec(
		"go run github.com/ysmood/golangci-lint@latest -- "+
			"run --no-config --fix --disable-all -E gofmt,goimports,misspell",
		path,
	)
}

func getDeviceList() gson.JSON {
	// we use the list from the web UI of devtools
	// TODO: We should keep update with their latest list, using hash id is a temp solution
	res, err := http.Get(
		"https://raw.githubusercontent.com/ChromeDevTools/devtools-frontend/c4e2fefe3327aa9fe5f4398a1baddb8726c230d5/front_end/emulated_devices/module.json",
	)
	utils.E(err)
	defer func() { _ = res.Body.Close() }()

	data, err := ioutil.ReadAll(res.Body)
	utils.E(err)

	return gson.New(data).Get("extensions")
}

func normalizeName(name string) string {
	name = strings.ReplaceAll(name, "/", "or")

	list := []string{}
	for _, s := range strings.Split(name, " ") {
		if len(s) > 1 {
			list = append(list, strings.ToUpper(s[0:1])+s[1:])
		} else {
			list = append(list, strings.ToUpper(s))
		}
	}

	return strings.Join(list, "")
}

func getUserAgent(val gson.JSON) string {
	ua := val.Get("user-agent").String()
	if ua == "" {
		return "Mozilla/5.0 (Macintosh; Intel Mac OS X 11_0_0) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/87.0.4280.88 Safari/537.36"
	}
	ua = strings.ReplaceAll(ua, "%s", "87.0.4280.88")
	return ua
}

func toGoArr(val gson.JSON) string {
	list := []string{}
	for _, s := range val.Arr() {
		list = append(list, s.String())
	}
	return fmt.Sprintf("%#v", list)
}
