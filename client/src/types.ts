// Bentuk data yang datang dari core.wasm.
//
// Ini adalah tipe untuk PRESENTASI, bukan salinan aturan. Perbedaan itu penting:
// menduplikasi bentuk data itu wajar dan murah, sedangkan menduplikasi LOGIKA
// aturan adalah hal yang secara eksplisit dilarang ADR-002. Tidak ada file di
// client yang boleh memutuskan apakah sebuah aksi legal, berapa resource yang
// didapat, atau kapan Darkness naik -- semua itu hanya ada di core.
//
// M1: tipe-tipe ini akan digenerate dari struct Go lewat cmd/contentgen
// (ADR-005), sehingga perubahan field tertangkap saat build, bukan saat runtime.

export type PlayerID = string;
export type LocationID = string;
export type ComponentID = string;
export type ObjectiveID = string;
export type CharacterID = string;

export type Resource = 'wood' | 'metal' | 'crystal' | 'food';

export const RESOURCES: Resource[] = ['wood', 'metal', 'crystal', 'food'];

/** ResourceSet di-serialisasi Go sebagai objek dengan nilai nol dihilangkan. */
export type ResourceSet = Partial<Record<Resource, number>>;

export type Status = 'lobby' | 'active' | 'won' | 'lost';
export type Phase = 'event' | 'player' | 'monster' | 'darkness';

export interface Deck {
  draw: string[];
  discard: string[];
}

/** Keputusan yang sedang ditunggu dari satu pemain (GDD §20). */
export interface PendingChoice {
  kind: 'mystery_option' | 'mystery_card';
  player: PlayerID;
  card?: string;
  cards?: string[] | null;
  options?: string[] | null;
}

export interface Player {
  id: PlayerID;
  name: string;
  character: CharacterID;
  at: LocationID;
  health: number;
  ap: number;
  inventory: ResourceSet;
  vp: number;
  exhausted: boolean;
  artifacts: string[] | null;
  /** Kosong untuk pemain lain: dikosongkan server saat projection (ADR-006). */
  objective: ObjectiveID;
  actedThisTurn: boolean;

  // Penghitung personal objective (GDD §24).
  explored: number;
  monstersSlain: number;
  villagesRescued: number;
  repairsJoined: number;
  resourcesGiven: number;
  wasExhausted: boolean;

  freeMoveUsed: boolean;
  abilityUsedTurn: boolean;
  repairDiscountUsed: boolean;
}

export interface GameLocation {
  id: LocationID;
  type: string;
  name: string;
  explored: boolean;
  adjacent: LocationID[];
  available: ResourceSet;
  monsters: number;
  gatherBlocked: boolean;
  rescued: boolean;
  investigated: boolean;
}

export interface Contribution {
  player: PlayerID;
  amount: number;
}

export interface Component {
  id: ComponentID;
  name: string;
  order: number;
  cost: ResourceSet;
  vp: number;
  repaired: boolean;
  progress: ResourceSet;
  contributions: Contribution[] | null;
}

export interface GameState {
  matchId: string;
  contentHash: string;
  rngState: number;
  status: Status;
  round: number;
  phase: Phase;
  turnOrder: PlayerID[];
  activeIdx: number;
  firstIdx: number;
  turnsTaken: number;
  players: Player[];
  board: { locations: GameLocation[] };
  darkness: number;
  lighthouse: Component[];

  /** Isi tumpukan tarik tampil sebagai '?' -- hanya jumlahnya yang publik. */
  eventDeck: Deck;
  mysteryDeck: Deck;
  artifactDeck: Deck;
  tileStack: string[];

  pending?: PendingChoice | null;
}

export interface PlayerView {
  viewer: PlayerID;
  state: GameState;
  myObjective?: ObjectiveID;
  hasObjective: PlayerID[] | null;
}

export type CommandKind =
  | 'move'
  | 'gather'
  | 'repair'
  | 'rest'
  | 'end_turn'
  | 'explore'
  | 'fight'
  | 'investigate'
  | 'trade'
  | 'choose';

export interface Command {
  kind: CommandKind;
  player: PlayerID;
  to?: LocationID;
  resource?: Resource;
  component?: ComponentID;
  pay?: ResourceSet;
  target?: PlayerID;
  give?: ResourceSet;
  option?: string;
  card?: string;
}

export interface GameEvent {
  kind: string;
  v: number;
  player?: PlayerID;
  from?: LocationID;
  to?: LocationID;
  resources?: ResourceSet;
  component?: ComponentID;
  objective?: ObjectiveID;
  amount?: number;
  value?: number;
  phase?: Phase;
  round?: number;
  reason?: string;
  deck?: string;
  card?: string;
  artifact?: string;
  option?: string;
  tile?: string;
  target?: PlayerID;
}

export function totalResources(rs: ResourceSet | undefined): number {
  if (!rs) return 0;
  return RESOURCES.reduce((sum, r) => sum + (rs[r] ?? 0), 0);
}
