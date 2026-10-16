//go:build windows

package bridge

import "golang.org/x/sys/windows/registry"

func registerKey(browser, path string) error {
	k, _, err := registry.CreateKey(registry.CURRENT_USER, `Software\`+browser+`\NativeMessagingHosts\`+HostName, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()
	return k.SetStringValue("", path)
}
