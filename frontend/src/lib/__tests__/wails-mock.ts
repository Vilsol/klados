import { vi } from 'vitest'

export const mockClusterService = {
  ListContexts: vi.fn().mockResolvedValue([]),
  Connect: vi.fn().mockResolvedValue(undefined),
  Disconnect: vi.fn().mockResolvedValue(undefined),
  ListNamespaces: vi.fn().mockResolvedValue([]),
  SwitchNamespace: vi.fn().mockResolvedValue(undefined),
  GetStatus: vi.fn().mockResolvedValue(0),
}

export const mockConfigService = {
  GetTheme: vi.fn().mockResolvedValue('system'),
  SetTheme: vi.fn().mockResolvedValue(undefined),
  GetConfig: vi.fn().mockResolvedValue(null),
  GetResolvedPrefs: vi.fn().mockResolvedValue({
    theme: 'system',
    accentColor: '',
    fontSize: 0,
    compactRows: false,
    readOnly: false,
    terminalWebGL: false,
    metrics: null,
    columnPrefs: null,
    favoriteNamespaces: null,
    keybindings: null,
    savedFilters: null,
  }),
  SetAccentColor: vi.fn().mockResolvedValue(undefined),
  SetFontSize: vi.fn().mockResolvedValue(undefined),
  SetStartupBehavior: vi.fn().mockResolvedValue(undefined),
  SetKeybinding: vi.fn().mockResolvedValue(undefined),
  ResetKeybindings: vi.fn().mockResolvedValue(undefined),
  GetClusterPrefs: vi.fn().mockResolvedValue(null),
  SetClusterPrefs: vi.fn().mockResolvedValue(undefined),
  DeleteClusterPrefs: vi.fn().mockResolvedValue(undefined),
  GetSavedFilters: vi.fn().mockResolvedValue([]),
  SetSavedFilters: vi.fn().mockResolvedValue(undefined),
  SetClusterSavedFilters: vi.fn().mockResolvedValue(undefined),
}

export const mockEvents = {
  On: vi.fn(),
  Off: vi.fn(),
  Emit: vi.fn(),
}

export const mockResourceService = {
  GetDescriptors: vi.fn().mockResolvedValue([]),
  ListResources: vi.fn().mockResolvedValue([]),
  GetResource: vi.fn().mockResolvedValue({}),
  DeleteResource: vi.fn().mockResolvedValue(undefined),
  UpdateResource: vi.fn().mockResolvedValue({}),
  ForceDeleteResource: vi.fn().mockResolvedValue(undefined),
  GetEvents: vi.fn().mockResolvedValue([]),
  ScaleResource: vi.fn().mockResolvedValue(undefined),
  RestartResource: vi.fn().mockResolvedValue(undefined),
  StartWatch: vi.fn().mockResolvedValue(undefined),
  StopWatch: vi.fn().mockResolvedValue(undefined),
  ListAPIResources: vi.fn().mockResolvedValue([]),
}

export const mockSchemaService = {
  GetSchema: vi.fn().mockResolvedValue({}),
}

export const mockPluginService = {
  ListPlugins: vi.fn().mockResolvedValue([]),
  GetPluginDescriptors: vi.fn().mockResolvedValue([]),
  GetPluginSidebarEntries: vi.fn().mockResolvedValue([]),
  GetPluginDetailTabs: vi.fn().mockResolvedValue([]),
  GetPluginCommands: vi.fn().mockResolvedValue([]),
  SaveRegistryCredentials: vi.fn().mockResolvedValue(undefined),
  AddInsecureRegistry: vi.fn().mockResolvedValue(undefined),
  GetPluginSettings: vi.fn().mockResolvedValue('{}'),
  SetPluginSettings: vi.fn().mockResolvedValue(undefined),
  GetPluginSettingsSchema: vi.fn().mockResolvedValue(''),
}

export function resetMocks() {
  Object.values(mockClusterService).forEach((fn) => fn.mockClear())
  Object.values(mockConfigService).forEach((fn) => fn.mockClear())
  Object.values(mockEvents).forEach((fn) => fn.mockClear())
  Object.values(mockResourceService).forEach((fn) => fn.mockClear())
  Object.values(mockSchemaService).forEach((fn) => fn.mockClear())
  Object.values(mockPluginService).forEach((fn) => fn.mockClear())
}
