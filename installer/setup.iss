; Astra-System Windows Installer
; Inno Setup Script — produces Astra-System-Setup.exe

#define MyAppName "Astra-System"
#define MyAppVersion "0.2.0"
#define MyAppPublisher "Astra-Service"
#define MyAppURL "https://github.com/astra-service/Astra-System"
#define MyAppExeName "astra-installer.exe"

[Setup]
AppId={{B8F4A9D2-3E7C-4F1A-9D6B-5C2E8A1F4B3D}
AppName={#MyAppName}
AppVersion={#MyAppVersion}
AppPublisher={#MyAppPublisher}
AppPublisherURL={#MyAppURL}
AppSupportURL={#MyAppURL}
AppUpdatesURL={#MyAppURL}
DefaultDirName={autopf}\{#MyAppName}
DefaultGroupName={#MyAppName}
DisableProgramGroupPage=yes
LicenseFile=..\LICENSE
OutputDir=..\dist
OutputBaseFilename=Astra-System-Setup
Compression=lzma2/ultra64
SolidCompression=yes
WizardStyle=modern
PrivilegesRequired=admin
MinVersion=10.0.19041
ArchitecturesInstallIn64BitMode=x64compatible
DisableWelcomePage=no
DisableFinishedPage=no
ShowLanguageDialog=no
LanguageDetectionMethod=none

[Languages]
Name: "english"; MessagesFile: "compiler:Default.isl"

[Tasks]
Name: "desktopicon"; Description: "Create a &desktop shortcut"; GroupDescription: "Additional icons:"; Flags: checkedonce

[Files]
; Binaries (built by CI from installer/bin/)
Source: "bin\astra-installer.exe"; DestDir: "{app}\bin"; Flags: ignoreversion
Source: "bin\astra-updater.exe"; DestDir: "{app}\bin"; Flags: ignoreversion

; Docker Compose files
Source: "..\docker-compose.prod.yml"; DestDir: "{app}\compose"; DestName: "docker-compose.yml"; Flags: ignoreversion
Source: "..\infra\nginx\kiosk.conf"; DestDir: "{app}\compose\nginx"; Flags: ignoreversion

; Config templates
Source: "resources\astra.conf.template"; DestDir: "{app}\config"; Flags: ignoreversion
Source: "resources\.env.template"; DestDir: "{app}"; DestName: ".env"; Flags: ignoreversion

; Documentation
Source: "..\README.md"; DestDir: "{app}"; Flags: ignoreversion

[Dirs]
Name: "{app}\bin"
Name: "{app}\compose"
Name: "{app}\config"
Name: "{app}\logs"

[Icons]
Name: "{group}\{#MyAppName}"; Filename: "{app}\bin\{#MyAppExeName}"; Parameters: "--channel stable"; WorkingDir: "{app}"
Name: "{group}\{#MyAppName} (Beta Channel)"; Filename: "{app}\bin\{#MyAppExeName}"; Parameters: "--channel beta"; WorkingDir: "{app}"
Name: "{group}\Astra-System Dashboard"; Filename: "http://localhost"; IconFilename: "{app}\bin\{#MyAppExeName}"
Name: "{group}\Uninstall {#MyAppName}"; Filename: "{uninstallexe}"
Name: "{commondesktop}\{#MyAppName}"; Filename: "{app}\bin\{#MyAppExeName}"; Parameters: "--channel stable"; WorkingDir: "{app}"; Tasks: desktopicon

[Run]
; Deploy Astra-System (pulls Docker images, starts services)
Filename: "{app}\bin\astra-installer.exe"; Parameters: "--install-dir ""{app}"" --data-dir ""{commonappdata}\Astra-System"" --silent"; Flags: runhidden runascurrentuser; StatusMsg: "Deploying Astra-System services (this may take several minutes)..."
; Open the kiosk dashboard
Filename: "http://localhost"; Flags: shellexec postinstall; Description: "Open Astra-System kiosk (after services are ready)"

[UninstallRun]
; Stop and remove the update agent service
Filename: "{app}\bin\astra-updater.exe"; Parameters: "remove"; Flags: runhidden runascurrentuser

[UninstallDelete]
Type: filesandordirs; Name: "{commonappdata}\Astra-System\config"
Type: filesandordirs; Name: "{commonappdata}\Astra-System\logs"
Type: filesandordirs; Name: "{commonappdata}\Astra-System\staging"
Type: filesandordirs; Name: "{commonappdata}\Astra-System\backups"
Type: filesandordirs; Name: "{commonappdata}\Astra-System\updates"

[Code]
function GetUninstallString: string;
var
  sUnInstPath: string;
  sUnInstallString: String;
begin
  sUnInstPath := ExpandConstant('Software\Microsoft\Windows\CurrentVersion\Uninstall\{#emit SetupSetting("AppId")}_is1');
  sUnInstallString := '';
  if not RegQueryStringValue(HKLM, sUnInstPath, 'UninstallString', sUnInstallString) then
    RegQueryStringValue(HKCU, sUnInstPath, 'UninstallString', sUnInstallString);
  Result := sUnInstallString;
end;

function IsUpgrade: Boolean;
begin
  Result := (GetUninstallString <> '');
end;

function GetPreviousDataDir(Value: string): string;
var
  uninstaller: string;
  params: string;
  idx: Integer;
begin
  uninstaller := GetUninstallString;
  if uninstaller = '' then
    Result := ''
  else begin
    { Extract --data-dir from the old uninstaller parameters if possible }
    idx := Pos('--data-dir ', uninstaller);
    if idx > 0 then
    begin
      params := Copy(uninstaller, idx + 11, Length(uninstaller));
      idx := Pos('"', params);
      if idx > 0 then
        Result := Copy(params, 1, idx - 1);
    end;
  end;
end;

function InitializeSetup: Boolean;
var
  uninstaller: string;
  ErrorCode: Integer;
begin
  if IsUpgrade then
  begin
    uninstaller := GetUninstallString;
    if uninstaller <> '' then
    begin
      if MsgBox('A previous version of Astra-System is installed. It will be uninstalled before continuing.', mbConfirmation, MB_OKCANCEL) = IDOK then
      begin
        Exec(RemoveQuotes(uninstaller), '/SILENT /NORESTART /SUPPRESSMSGBOXES', '', SW_HIDE, ewWaitUntilTerminated, ErrorCode);
      end
      else
      begin
        Result := False;
        Exit;
      end;
    end;
  end;
  Result := True;
end;
