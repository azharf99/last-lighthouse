import type { Command, PendingChoice, PlayerID } from '../types';
import { MYSTERY_CARDS } from './labels';

interface Props {
  pending: PendingChoice;
  viewer: PlayerID;
  playerName: string;
  legal: Command[];
  disabled: boolean;
  onSend(cmd: Command): void;
}

/**
 * Dialog kartu Mystery (GDD §20).
 *
 * Ditampilkan sebagai overlay yang menghalangi aksi lain karena aturannya memang
 * begitu: selama pilihan belum dijawab, core menolak command apa pun selain
 * jawabannya. UI yang membiarkan pemain mengklik hal lain hanya akan menghasilkan
 * penolakan beruntun.
 *
 * Yang ditampilkan hanya TEKS pilihannya, bukan efeknya. Itu disengaja dan
 * merupakan inti mekaniknya: "pemain harus memutuskan apakah imbalannya sepadan
 * dengan risikonya" (GDD §20). Kalau hasilnya sudah tertulis di tombol, tidak
 * ada lagi yang perlu diputuskan.
 */
export function MysteryDialog({
  pending,
  viewer,
  playerName,
  legal,
  disabled,
  onSend,
}: Props) {
  const mine = pending.player === viewer;

  if (!mine) {
    return (
      <div className="overlay">
        <div className="overlay__card">
          <h2 className="overlay__title">Menunggu keputusan</h2>
          <p className="overlay__text">
            <b>{playerName}</b> sedang menghadapi sebuah misteri.
          </p>
        </div>
      </div>
    );
  }

  // Tahap 1 (kemampuan Scholar, GDD §10.4): pilih satu dari dua kartu.
  if (pending.kind === 'mystery_card') {
    const cards = pending.cards ?? [];
    return (
      <div className="overlay">
        <div className="overlay__card overlay__card--wide">
          <h2 className="overlay__title">Pengetahuan Kuno</h2>
          <p className="overlay__text">
            Kau menarik dua misteri. Pilih satu untuk dihadapi; yang lain dibuang.
          </p>
          <div className="choices">
            {cards.map((id) => {
              const card = MYSTERY_CARDS[id];
              const cmd: Command = legal.find((l) => l.card === id) ?? {
                kind: 'choose',
                player: viewer,
                card: id,
              };
              return (
                <button
                  key={id}
                  className="choice"
                  disabled={disabled}
                  onClick={() => onSend(cmd)}
                >
                  <span className="choice__title">{card?.name ?? id}</span>
                  <span className="choice__text">{card?.text ?? ''}</span>
                </button>
              );
            })}
          </div>
        </div>
      </div>
    );
  }

  // Tahap 2: pilih salah satu opsi kartu.
  const card = pending.card ? MYSTERY_CARDS[pending.card] : undefined;
  const available = new Set(pending.options ?? []);

  return (
    <div className="overlay">
      <div className="overlay__card overlay__card--wide">
        <h2 className="overlay__title">{card?.name ?? 'Misteri'}</h2>
        <p className="overlay__text">{card?.text ?? ''}</p>

        <div className="choices">
          {(card?.options ?? []).map((opt) => {
            // Pilihan yang tidak terjangkau (mis. tidak mampu membayar) sudah
            // disaring core; di sini ia tetap ditampilkan tapi mati, supaya
            // pemain melihat bahwa pilihan itu ADA dan tahu apa yang ia lewatkan.
            const usable = available.has(opt.id);
            const cmd: Command = legal.find((l) => l.option === opt.id) ?? {
              kind: 'choose',
              player: viewer,
              option: opt.id,
            };
            return (
              <button
                key={opt.id}
                className="choice"
                disabled={disabled || !usable}
                onClick={() => usable && onSend(cmd)}
              >
                <span className="choice__title">
                  {opt.id.toUpperCase()} — {opt.text}
                </span>
                {!usable && (
                  <span className="choice__text">Tidak terjangkau saat ini</span>
                )}
              </button>
            );
          })}
        </div>

        <p className="tiny muted" style={{ marginTop: 14, marginBottom: 0 }}>
          Hasilnya baru terlihat setelah kau memilih.
        </p>
      </div>
    </div>
  );
}
