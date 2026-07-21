; Astra-System Windows Installer — Inno Setup wrapper
; This script ONLY places files and creates shortcuts.
; The actual installation wizard runs when you open the shortcut.

#define AppName "Astra-System"
#define AppVersion "0.2.0"
#define AppPublisher "Astra-Service"
#define AppURL "https://github.com/xdfkenny/Astra-System"

[Setup]
AppId={{A8F4A9D2-3E7C-4F1A-9D6B-5C2E8A1F4B3D}
AppName={#AppName}
AppVersion={#AppVersion}
AppPublisher={#AppPublisher}
AppPublisherURL={#AppURL}
AppSupportURL={#AppURL}
AppUpdatesURL={#AppURL}
DefaultDirName={autopf}\{#AppName}
DefaultGroupName={#AppName}
DisableProgramGroupPage=yes
OutputDir=..\dist
OutputBaseFilename=Astra-System-Setup
Compression=lzma2/ultra64
SolidCompression=yes
WizardStyle=modern
PrivilegesRequired=admin
MinVersion=10.0.19041
ArchitecturesInstallIn64BitMode=x64compatible
UninstallDisplayName={#AppName}

[Languages]
Name: "english"; MessagesFile: "compiler:Default.isl"

[Tasks]
Name: "desktopicon"; Description: "Create a &desktop shortcut"; GroupDescription: "Additional:"

[Files]
Source: "bin\astra-installer.exe"; DestDir: "{app}\bin"; Flags: ignoreversion
Source: "bin\astra-updater.exe"; DestDir: "{app}\bin"; Flags: ignoreversion

[Dirs]
Name: "{app}\bin"

[Icons]
Name: "{group}\{#AppName}"; Filename: "{app}\bin\astra-installer.exe"; WorkingDir: "{app}"
Name: "{group}\Uninstall {#AppName}"; Filename: "{uninstallexe}"
Name: "{commondesktop}\{#AppName}"; Filename: "{app}\bin\astra-installer.exe"; WorkingDir: "{app}"; Tasks: desktopicon

[Run]
Filename: "{app}\bin\astra-installer.exe"; Description: "Launch Astra-System setup wizard"; Flags: postinstall skipifsilent

[UninstallRun]
Filename: "{app}\bin\astra-updater.exe"; Parameters: "remove"; Flags: runascurrentuser

[UninstallDelete]
Type: filesandordirs; Name: "{commonappdata}\Astra-System"
