export interface Application {
  name: string;
  path: string;
  icon?: string;
  allowChildren: boolean;
}
export interface Website {
  domain: string;
  subdomains: boolean;
  resources?: string[];
}
export interface Profile {
  id: string;
  name: string;
  applications: Application[];
  websites: Website[];
  mode: "strict" | "blocklist";
}
export interface Settings {
  language: "ru" | "en";
  exitDelaySeconds: number;
  onboarded: boolean;
}
export interface KotlinQuestion {
  token: string;
  prompt: string;
  options: string[];
  sourceUrl: string;
}
export interface StudyPlan {
  token: string;
  blocked: { name: string; path: string }[];
  browserReady: boolean;
  docsUrl: string;
}
export interface Session {
  exercise: boolean;
  elapsedSeconds: number;
  state: "IDLE" | "ACTIVE" | "EXIT_PENDING" | "FINISHED";
  id: string;
  profile: Profile;
  remainingSeconds: number;
  exitRemainingSeconds: number;
  durationSeconds: number;
  progress: number;
  outcome: string;
  appBlocks: number;
  siteBlocks: number;
}
export interface Record {
  id: string;
  profile: string;
  startedAt: string;
  endedAt: string;
  seconds: number;
  outcome: string;
  appBlocks: number;
  siteBlocks: number;
}
export interface Statistics {
  seconds: number;
  completed: number;
  interrupted: number;
  appBlocks: number;
  siteBlocks: number;
  days: { date: string; seconds: number }[];
  history: Record[];
}
export interface State {
  session: Session;
  settings: Settings;
  profiles: Profile[];
  browsers: {
    browser: string;
    path: string;
    pid: number;
    connected: boolean;
    resourceCapable: boolean;
    applied: string;
  }[];
  statistics: Statistics;
  preview: boolean;
  error: string;
}
export type Page =
  | "dashboard"
  | "applications"
  | "websites"
  | "profiles"
  | "statistics"
  | "settings";
