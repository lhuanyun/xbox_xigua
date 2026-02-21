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
	XINPUT_GAMEPAD_LEFT_THUMB = 0x0040 // L3 键（左摇杆按下）
	ERROR_SUCCESS             = 0
)

var xinputGetEx *syscall.Proc

// 智能加载 DLL 的函数保持不变
func initXInput() error {
	dllNames := []string{
		"xinput1_4.dll",
		"xinput1_3.dll",
		"xinput1_2.dll",
		"xinput1_1.dll",
		"xinput9_1_0.dll",
	}

	for _, name := range dllNames {
		dll, err := syscall.LoadDLL(name)
		if err == nil {
			proc, err := dll.FindProc("#100")
			if err == nil {
				xinputGetEx = proc
				fmt.Printf("成功加载手柄驱动: %s\n", name)
				return nil
			}
			dll.Release()
		}
	}
	return fmt.Errorf("你的系统缺少所有版本的 Xbox 手柄驱动 (XInput DLL)")
}

func main() {
	err := initXInput()
	if err != nil {
		fmt.Println("启动失败:", err)
		time.Sleep(10 * time.Second)
		return
	}

	var state XINPUT_STATE
	var pressStartTime time.Time
	
	isPressing := false
	hasTriggered := false

	fmt.Println("正在后台监听 L3 键 (左摇杆按下)...")

	for {
		if xinputGetEx != nil {
			ret, _, _ := xinputGetEx.Call(0, uintptr(unsafe.Pointer(&state)))

			if ret == ERROR_SUCCESS {
				// 检查 L3 键是否被按下
				isL3Pressed := (state.Gamepad.wButtons & XINPUT_GAMEPAD_LEFT_THUMB) != 0

				if isL3Pressed {
					if !isPressing {
						isPressing = true
						hasTriggered = false
						pressStartTime = time.Now()
					} else if !hasTriggered {
						if time.Since(pressStartTime) >= 10*time.Second {
							// 长按达到 10 秒，触发读取文件并执行
							go launchPs1FromConf()
							hasTriggered = true
						}
					}
				} else {
					isPressing = false
					hasTriggered = false
				}
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// 动态读取 command.conf 并执行
func launchPs1FromConf() {
	// 1. 获取当前 exe 所在的完整路径
	exePath, err := os.Executable()
	if err != nil {
		fmt.Println("无法获取当前程序路径:", err)
		return
	}
	
	// 2. 获取 exe 所在的目录文件夹
	exeDir := filepath.Dir(exePath)
	
	// 3. 拼接 command.conf 的绝对路径
	confPath := filepath.Join(exeDir, "command.conf")

	// 4. 读取文件内容
	content, err := os.ReadFile(confPath)
	if err != nil {
		fmt.Printf("读取 command.conf 失败 (请确认文件存在): %v\n", err)
		return
	}

	// 5. 清理文件内容中多余的空格或回车换行符
	commandStr := strings.TrimSpace(string(content))
	if commandStr == "" {
		fmt.Println("command.conf 文件内容为空！")
		return
	}

	// 6. 使用 PowerShell 隐藏执行读取到的命令
	// 这里使用了 -Command 参数，所以配置文件里可以直接写 .ps1 的路径，也可以直接写 PowerShell 原生指令
	cmd := exec.Command("powershell.exe", "-ExecutionPolicy", "Bypass", "-WindowStyle", "Hidden", "-Command", commandStr)
	err = cmd.Start()
	if err != nil {
		fmt.Printf("执行 PowerShell 失败: %v\n", err)
	} else {
		fmt.Println("成功触发命令:", commandStr)
	}
}
