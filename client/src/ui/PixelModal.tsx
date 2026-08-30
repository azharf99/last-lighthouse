import { useEffect, type ReactNode } from 'react';

interface Props {
  isOpen: boolean;
  onClose(): void;
  title: string;
  children: ReactNode;
  wide?: boolean;
}

/**
 * Retro pixel-art styled modal overlay.
 * Uses double-border pixel aesthetic like classic RPG menu windows.
 */
export function PixelModal({ isOpen, onClose, title, children, wide }: Props) {
  useEffect(() => {
    if (!isOpen) return;
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        e.preventDefault();
        onClose();
      }
    };
    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [isOpen, onClose]);

  if (!isOpen) return null;

  return (
    <div className="pixel-overlay" onClick={onClose}>
      <div
        className={`pixel-modal ${wide ? 'pixel-modal--wide' : ''}`}
        onClick={(e) => e.stopPropagation()}
      >
        <div className="pixel-modal__header">
          <span className="pixel-modal__title">{title}</span>
          <button
            className="pixel-modal__close"
            onClick={onClose}
            aria-label="Tutup"
          >
            ✕
          </button>
        </div>
        <div className="pixel-modal__body">
          {children}
        </div>
      </div>
    </div>
  );
}
