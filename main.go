package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

type XINPUT_GAMEPAD struct {
	wButtons      uint16
	bLeftTrigger  byte
	bRightTrigger byte
	sThumbLX      int16
	sThumbLY      int16
	sThumbRX      int16
	sThumbRY      int16
}

type XINPUT_STATE struct {
	dwPacketNumber uint32
	Gamepad        XINPUT_GAMEPAD
}

const (
	XINPUT_GAMEPAD_LEFT_THUMB = 0x0040 // 标准的 L3 键掩码
	ERROR_SUCCESS             = 0
)

var xinputGetState *syscall.Proc

// 写日志助手：把运行状态悄悄写进 error_log.txt，方便排错
func writeLog(msg string) {
	exePath, err := os.Executable()
	if err != nil {
		return
	}
	logPath := filepath.Join(filepath.Dir(exePath), "error_log.txt")
	// 追加模式写入日志，带上时间戳
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err == nil {
		f.WriteString(time.Now().Format("2006-01-02 15:04:05") + " - " + msg + "\n")
		f.Close()
	}
}

// 智能加载稳定版的官方驱动
func initXInput() error {
	dllNames := []string{"xinput1_4.dll", "xinput1_3.dll", "xinput9_1_0.dll"}

	for _, name := range dllNames {
		dll, err := syscall.LoadDLL(name)
		if err == nil {
			// L3 是标准按键，直接使用官方公开的接口 XInputGetState 即可，100% 稳定
			proc, err := dll.FindProc("XInputGetState")
			if err == nil {
				xinputGetState = proc
				writeLog("成功加载手柄驱动: " + name)
				return nil
			}
			dll.Release()
		}
	}
	return fmt.Errorf("系统缺少手柄驱动 DLL")
}

func main() {
	err := initXInput()
	if err != nil {
		writeLog("启动失败: " + err.Error())
		return // 加载失败才会退出
	}

	var state XINPUT_STATE
	var pressStartTime time.Time
	
	isPressing := false
	hasTriggered := false

	writeLog("程序已成功驻留后台，正在持续监听 L3...")

	for {
		if xinputGetState != nil {
			ret, _, _ := xinputGetState.Call(0, uintptr(unsafe.Pointer(&state)))

			if ret == ERROR_SUCCESS {
				// 手柄已连接且工作正常
				isL3Pressed := (state.Gamepad.wButtons & XINPUT_GAMEPAD_LEFT_THUMB) != 0

				if isL3Pressed {
					if !isPressing {
						isPressing = true
						hasTriggered = false
						pressStartTime = time.Now()
					} else if !hasTriggered {
						if time.Since(pressStartTime) >= 6*time.Second {
							go launchPs1FromConf()
							hasTriggered = true
						}
					}
				} else {
					isPressing = false
					hasTriggered = false
				}
			} else {
				// 如果手柄休眠或被拔掉，重置状态，防止按键状态卡死
				isPressing = false
				hasTriggered = false
			}
		}
		// 降低 CPU 占用，50 毫秒轮询一次足够灵敏
		time.Sleep(500 * time.Millisecond)
	}
}

func launchPs1FromConf() {
	exePath, err := os.Executable()
	if err != nil {
		writeLog("执行失败：无法获取当前 exe 路径")
		return
	}
	
	confPath := filepath.Join(filepath.Dir(exePath), "command.conf")

	content, err := os.ReadFile(confPath)
	if err != nil {
		writeLog("触发中止：找不到或无法读取 command.conf 文件")
		return
	}

	commandStr := strings.TrimSpace(string(content))
	if commandStr == "" {
		writeLog("触发中止：command.conf 里没有写入任何命令")
		return
	}

	cmd := exec.Command("powershell.exe", "-ExecutionPolicy", "Bypass", "-WindowStyle", "Hidden", "-Command", commandStr)
	err = cmd.Start()
	if err != nil {
		writeLog("执行 PowerShell 失败: " + err.Error())
	} else {
		writeLog("成功触发！已静默执行命令: " + commandStr)
	}
}
