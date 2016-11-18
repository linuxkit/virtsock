#$MsgSzs = 1024, 2048
$MsgSzs = 8, 64, 128, 512, 1024, 1518, 2048, 4096, 8192, 16384

# collect some info
$winver = (Get-ItemProperty -Path c:\windows\system32\hal.dll).VersionInfo.FileVersion
$cpuinfo = (Get-WmiObject Win32_Processor).Name
$mem = (Get-WmiObject -Class Win32_ComputerSystem).TotalPhysicalMemory
$mem = [Math]::Round(($mem/ 1GB),2)
$info = (docker info)
$linver = $info | select-string -pattern "Kernel"
$vmmem = $info | select-string -pattern "Memory"
$vmcpus = $info | select-string -pattern "CPUs"
Write-Output "# Windows Version: $winver"
Write-Output "# CPU: $cpuinfo"
Write-Output "# Memory: $mem GB"
Write-Output "# Linux $linver"
Write-Output "# Linux $vmmem"
Write-Output "# Linux $vmcpus"

# get current path as something  we can use inside the VM
$CurDir = (& 'C:\Program Files\Git\usr\bin\cygpath.exe' $pwd)
$VMId = (Get-VM MobyLinuxVM).Id

# preload the docker nsenter image
docker pull justincormack/nsenter1 > $null
# Copy hvbench to a more convenient location
docker run --rm -ti --privileged --pid=host justincormack/nsenter1 /bin/cp $CurDir/hvbench /

# Can't -RedirectStandardError to $null. Create a dummy file...
$errout = ".\hvbench.err.txt"

Write-Output "# Loopback mode"
foreach ($MsgSz in $MsgSzs) {
    # Start the server on the host and give it time to start
    Start-Process -NoNewWindow -FilePath .\hvbench.exe -ArgumentList "-s -b"  -RedirectStandardError $errout 
    Start-Sleep -s 2
    .\hvbench.exe -c loopback -b -m $MsgSz
}

Write-Output ""
Write-Output ""
Write-Output "# Transmit from VM"
foreach ($MsgSz in $MsgSzs) {
    # Start the server on the host and give it time to start
    Start-Process -NoNewWindow -FilePath .\hvbench.exe -ArgumentList "-s -b" -RedirectStandardError $errout 
    Start-Sleep -s 2
    docker run --rm --privileged --pid=host justincormack/nsenter1 /hvbench -c parent -b -m $MsgSz
}


if ($linver.ToString() -match "4.4") {
    Write-Output "# Transmit to VM skipped to Linux kernel $linver"
    return
}

# We only have 4.4 or later. For later kernels run the other direction to
Write-Output ""
Write-Output ""
Write-Output "# Transmit to VM"
foreach ($MsgSz in $MsgSzs) {
    # Start the server in the VM detached
    $svrid = docker run -d --privileged --pid=host justincormack/nsenter1 /hvbench -s -b
    Start-Sleep -s 2
    .\hvbench.exe -c $VMId -b -m $MsgSz
    docker kill $svrid 2> $null
}