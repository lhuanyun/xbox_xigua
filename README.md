-Command 的灵活性：我把 PowerShell 的调用参数改成了 -Command。这让你的 command.conf 用法变得非常灵活：

用法 A（直接写路径）：文件里只写一行：C:\scripts\my_script.ps1

用法 B（写带参数的脚本）：文件里写一行：C:\scripts\my_script.ps1 -Force

用法 C（写原生指令）：文件里写一行：Stop-Process -Name "notepad" -Force

你可以把这段代码推送到 GitHub 上进行自动编译了。编译完成后，记得在解压出来的 .exe 旁边，自己新建一个名为 command.conf 的文本文件。
