package main

import (
	"runtime"
	"testing"

	"github.com/wailsapp/wails/v2/pkg/menu"
)

func TestNativeMenuKeepsMacOSEditCommands(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("native menu roles are specific to macOS")
	}

	items := buildNativeMenu(NewApp()).Items
	wantRoles := map[menu.Role]bool{
		menu.AppMenuRole:    false,
		menu.EditMenuRole:   false,
		menu.WindowMenuRole: false,
	}
	for _, item := range items {
		if _, tracked := wantRoles[item.Role]; tracked {
			wantRoles[item.Role] = true
		}
	}
	for role, found := range wantRoles {
		if !found {
			t.Errorf("native menu role %d is missing", role)
		}
	}
}
