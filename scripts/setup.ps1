$service = New-Item -Path "HKLM:\SOFTWARE\Microsoft\Windows NT\CurrentVersion\Virtualization\GuestCommunicationServices" -Name 3049197C-9A4E-4FBF-9367-97F792F16994

$service.SetValue("ElementName", "Hyper-V Socket Play Service")
