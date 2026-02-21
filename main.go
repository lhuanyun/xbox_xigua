package main

import (
	"fmt"
	"os/exec"
	"syscall"
	"time"
	"unsafe"
)

// 定义 XInput 结构体，用于接收 DLL 返回的手柄状态数据
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
	XINPUT_GAMEPAD_GUIDE = 0x0400 // 西瓜键(Guide Button)的隐藏掩码
	ERROR_SUCCESS        = 0
)

var (
	// 加载 Windows 10/11 默认的手柄驱动库
	xinput = syscall.NewLazyDLL("xinput1_4.dll")
	// 通过序号 100 获取隐藏的 XInputGetStateEx 函数
	xinputGetEx = xinput.NewProc("#100")
)

func main() {
	var state XINPUT_STATE
	var pressStartTime time.Time
	
	isPressing := false
	hasTriggered := false

	// 要运行的 exe 程序路径 (请根据需要修改)
	// 注意：Go 语言中反斜杠需要转义，例如 "C:\\Tools\\MyApp.exe"
	targetExePath := "calc.exe" // 默认用系统计算器做演示

	fmt.Println("正在后台监听西瓜键...")

	for {
		// 轮询第一个手柄 (索引为 0)
		ret, _, _ := xinputGetEx.Call(0, uintptr(unsafe.Pointer(&state)))

		if ret == ERROR_SUCCESS {
			// 检查西瓜键是否被按下
			isGuidePressed := (state.Gamepad.wButtons & XINPUT_GAMEPAD_GUIDE) != 0

			if isGuidePressed {
				if !isPressing {
					// 刚刚按下的一瞬间
					isPressing = true
					hasTriggered = false
					pressStartTime = time.Now()
				} else if !hasTriggered {
					// 持续按下中，检查是否达到 10 秒
					if time.Since(pressStartTime) >= 10*time.Second {
						// 触发运行 EXE
						go launchApp(targetExePath)
						hasTriggered = true // 标记已触发，防止按住不放疯狂运行
					}
				}
			} else {
				// 按键松开，重置状态
				isPressing = false
				hasTriggered = false
			}
		}

		// 休眠 50 毫秒 (一秒检测 20 次)，避免占用过多 CPU 性能
		time.Sleep(50 * time.Millisecond)
	}
}

func launchApp(path string) {
	cmd := exec.Command(path)
	err := cmd.Start() // 使用 Start 而不是 Run，这样不会阻塞当前手柄的监听进程
	if err != nil {
		fmt.Printf("无法运行程序: %v\n", err)
	}
}
