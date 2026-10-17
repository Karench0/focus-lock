#ifndef AppVersion
  #define AppVersion "0.3.2"
#endif
[Setup]
AppId={{3A3EE70B-92CE-450E-9B1D-C266FF5EF10C}
AppName=Focus Lock
AppVersion={#AppVersion}
AppPublisher=Karen Mkrtchyan
AppPublisherURL=https://github.com/Karench0/focus-lock
DefaultDirName={localappdata}\Programs\FocusLock
DefaultGroupName=Focus Lock
PrivilegesRequired=lowest
ArchitecturesAllowed=x64compatible
ArchitecturesInstallIn64BitMode=x64compatible
OutputDir=..\build\release
OutputBaseFilename=FocusLock-{#AppVersion}-setup
SetupIconFile=..\build\windows\icon.ico
UninstallDisplayIcon={app}\FocusLock.exe
Compression=lzma2
SolidCompression=yes
WizardStyle=modern
CloseApplications=yes
RestartApplications=no
LicenseFile=..\LICENSE

[Files]
Source: "..\build\bin\FocusLock.exe"; DestDir: "{app}"; Flags: ignoreversion
Source: "..\build\bin\FocusLock-host.exe"; DestDir: "{app}"; Flags: ignoreversion
Source: "..\browser-extension\*"; DestDir: "{app}\browser-extension"; Flags: ignoreversion recursesubdirs createallsubdirs; Excludes: "tests\*"
Source: "..\LICENSE"; DestDir: "{app}"; Flags: ignoreversion
Source: "..\README.md"; DestDir: "{app}"; Flags: ignoreversion
Source: "..\THIRD_PARTY_NOTICES.md"; DestDir: "{app}"; Flags: ignoreversion
Source: "..\third-party-licenses\*"; DestDir: "{app}\third-party-licenses"; Flags: ignoreversion recursesubdirs createallsubdirs
Source: "..\docs\*"; DestDir: "{app}\docs"; Flags: ignoreversion recursesubdirs createallsubdirs

[Icons]
Name: "{group}\Focus Lock"; Filename: "{app}\FocusLock.exe"
Name: "{group}\Uninstall Focus Lock"; Filename: "{uninstallexe}"

[UninstallDelete]
Type: files; Name: "{app}\focus-lock-native-host.json"

[Code]
procedure RemoveOwnedHost(const Key: String);
var
  ManifestPath: String;
begin
  if RegQueryStringValue(HKCU, Key, '', ManifestPath) then
    if CompareText(ManifestPath, ExpandConstant('{app}\focus-lock-native-host.json')) = 0 then
      RegDeleteKeyIncludingSubkeys(HKCU, Key);
end;

procedure CurUninstallStepChanged(CurUninstallStep: TUninstallStep);
begin
  if CurUninstallStep = usUninstall then
  begin
    RemoveOwnedHost('Software\Google\Chrome\NativeMessagingHosts\com.focuslock.bridge');
    RemoveOwnedHost('Software\Microsoft\Edge\NativeMessagingHosts\com.focuslock.bridge');
  end;
end;
