import { useState } from 'react';
import { sfx } from '../audio/sfx';

interface Props {
  isOpen: boolean;
  onClose: () => void;
}

export function TutorialModal({ isOpen, onClose }: Props) {
  const [slide, setSlide] = useState(0);

  if (!isOpen) return null;

  const slides = [
    {
      title: '🌟 Misi Utama: Nyalakan Mercusuar Terakhir',
      content: (
        <div>
          <p>
            Kabut kegelapan (<b>The Darkness</b>) merayap menyelimuti pulau. Tujuan tim adalah menyalakan
            <b> 5 Komponen Mercusuar</b> (Lensa, Bahan Bakar, Gearbox, Prisma, dan Generator) sebelum
            Darkness mencapai angka <b>8</b>.
          </p>
          <div
            style={{
              padding: 12,
              background: 'rgba(255, 199, 107, 0.08)',
              border: '1px solid var(--beacon)',
              borderRadius: 6,
              fontSize: 13,
            }}
          >
            💡 <b>Kerja Sama Kooperatif:</b> Seluruh pemain menang bersama saat ke-5 komponen menyala,
            atau kalah bersama jika Darkness mencapai batas maksimal!
          </div>
        </div>
      ),
    },
    {
      title: '🧭 4 Peran Asimetris dengan Keahlian Unik',
      content: (
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 10, fontSize: 13 }}>
          <div style={{ padding: 10, background: 'var(--bg-card)', borderRadius: 6, border: '1px solid var(--stone)' }}>
            <b>🧭 The Navigator</b>
            <p className="tiny muted" style={{ marginTop: 4 }}>
              Dapat bergerak 2 langkah per 1 AP dan mengurangi kabut di lokasi tetangga.
            </p>
          </div>
          <div style={{ padding: 10, background: 'var(--bg-card)', borderRadius: 6, border: '1px solid var(--stone)' }}>
            <b>⚙️ The Engineer</b>
            <p className="tiny muted" style={{ marginTop: 4 }}>
              Memperbaiki komponen mercusuar dengan diskon 1 sumber daya dan efisiensi crafting tinggi.
            </p>
          </div>
          <div style={{ padding: 10, background: 'var(--bg-card)', borderRadius: 6, border: '1px solid var(--stone)' }}>
            <b>🏹 The Hunter</b>
            <p className="tiny muted" style={{ marginTop: 4 }}>
              Ahli bertarung melawan Abyssal Beast (+1 bonus roll dadu tempur) dan melindungi rekan.
            </p>
          </div>
          <div style={{ padding: 10, background: 'var(--bg-card)', borderRadius: 6, border: '1px solid var(--stone)' }}>
            <b>📜 The Scholar</b>
            <p className="tiny muted" style={{ marginTop: 4 }}>
              Mendekripsi rahasia pulau, mengendalikan kartu peristiwa, dan menahan laju Darkness.
            </p>
          </div>
        </div>
      ),
    },
    {
      title: '⚡ Sistem 3 Action Points (AP) per Giliran',
      content: (
        <div style={{ fontSize: 13 }}>
          <p>Setiap giliran, Anda memiliki <b>3 AP</b> untuk melakukan tindakan:</p>
          <ul style={{ paddingLeft: 20, display: 'flex', flexDirection: 'column', gap: 6 }}>
            <li><b>🏃 Bergerak (1 AP):</b> Berpindah ke lokasi yang terhubung di peta.</li>
            <li><b>🪓 Mengumpulkan (1 AP):</b> Mengambil sumber daya alam (Kayu, Logam, Kristal).</li>
            <li><b>🔨 Memperbaiki (1 AP):</b> Memperbaiki komponen mercusuar jika memiliki bahan.</li>
            <li><b>⚔️ Bertarung (1 AP):</b> Melawan monster dengan lemparan dadu 1D6!</li>
          </ul>
        </div>
      ),
    },
    {
      title: '🎲 Pertarungan Dadu 1D6 & Ambang Kegelapan',
      content: (
        <div>
          <p style={{ fontSize: 13 }}>
            Saat bertarung melawan monster, lempar dadu <b>1D6</b>. Hasil lemparan ditambah bonus peran menentukan
            apakah Anda berhasil melukai monster atau terkena serangan balasan.
          </p>
          <div
            style={{
              padding: 12,
              background: 'rgba(235, 87, 87, 0.08)',
              border: '1px solid var(--dread)',
              borderRadius: 6,
              fontSize: 13,
            }}
          >
            ⚠️ <b>Awas Kegelapan:</b> Di akhir setiap ronde penuh, Darkness naik 1 tingkat dan monster baru dapat
            muncul dari jurang berkabut!
          </div>
        </div>
      ),
    },
  ];

  const currentSlide = slides[slide];

  return (
    <div className="overlay" style={{ zIndex: 1100 }} data-testid="tutorial-modal">
      <div className="overlay__card" style={{ maxWidth: 620, width: '92%' }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
          <h2 className="overlay__title" style={{ margin: 0 }}>
            📖 Panduan Bermain (Tutorial)
          </h2>
          <button
            className="action action--ghost"
            onClick={() => {
              sfx.playClick();
              onClose();
            }}
            aria-label="Tutup Panduan"
          >
            ✕
          </button>
        </div>

        {/* Slide Title */}
        <h3 style={{ fontSize: 16, color: 'var(--beacon)', marginBottom: 12 }}>
          {currentSlide.title}
        </h3>

        {/* Slide Body */}
        <div style={{ minHeight: 180, marginBottom: 20 }}>
          {currentSlide.content}
        </div>

        {/* Navigation Dots & Buttons */}
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <button
            className="action"
            onClick={() => {
              sfx.playClick();
              setSlide((prev) => Math.max(0, prev - 1));
            }}
            disabled={slide === 0}
            aria-label="Slide Panduan Sebelumnya"
          >
            ◀️ Sebelumnya
          </button>

          {/* Dots Indicator */}
          <div style={{ display: 'flex', gap: 6 }}>
            {slides.map((_, idx) => (
              <div
                key={idx}
                onClick={() => {
                  sfx.playClick();
                  setSlide(idx);
                }}
                style={{
                  width: 8,
                  height: 8,
                  borderRadius: '50%',
                  background: idx === slide ? 'var(--beacon)' : 'var(--stone)',
                  cursor: 'pointer',
                }}
              />
            ))}
          </div>

          {slide < slides.length - 1 ? (
            <button
              className="action action--primary"
              onClick={() => {
                sfx.playClick();
                setSlide((prev) => Math.min(slides.length - 1, prev + 1));
              }}
              aria-label="Slide Panduan Selanjutnya"
              data-testid="tutorial-btn-next"
            >
              Selanjutnya ▶️
            </button>
          ) : (
            <button
              className="action action--primary"
              onClick={() => {
                sfx.playClick();
                onClose();
              }}
              data-testid="tutorial-btn-finish"
            >
              🚀 Siap Bermain!
            </button>
          )}
        </div>
      </div>
    </div>
  );
}
