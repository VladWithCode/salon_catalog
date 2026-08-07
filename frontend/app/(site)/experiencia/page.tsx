import type { Metadata } from "next";
import Image from "next/image";

// Copy migrated verbatim from internal/templates/pages/salon.templ.
// The Go page's video-tour popup and gallery lightbox (popup.js + GSAP,
// internal/templates/pages/salon.templ's salonScript) are not reproduced:
// this page uses a native <video controls> (playable without any
// JavaScript, no autoplay, matches section 17's "sin autoplay" and "sin
// Motion obligatorio") and a plain image grid where each photo opens at
// full size via a real link — no lightbox, per the explicit permission to
// skip one "si aumenta demasiado el alcance".
export const metadata: Metadata = {
  title: "Experiencia",
  description:
    "Conoce el salón de eventos Villa Chenacolo: arquitectura, recorrido en video y galería de fotos.",
};

// Same asset set internal/templates/pages/salon.templ's allGalleryImages
// uses, already synced to frontend/public/assets (see
// frontend/scripts/sync-static-assets.mjs).
const galleryImages = [
  1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21,
  22, 24, 25, 26, 27, 28, 29, 30, 31,
].map((n) => `/assets/chenacolo_${n}.jpeg`);

export default function ExperiencePage() {
  return (
    <article>
      <section className="section-light py-section">
        <div className="container-main max-w-3xl space-y-4 text-center">
          <h1 className="type-h1 font-medium">Conoce nuestro salón</h1>
          <p className="type-lead text-muted-foreground">
            Chenacolo es un salón de eventos innovador y personalizado,
            diseñado para ofrecer una experiencia de eventos única.
          </p>
        </div>
      </section>

      <section className="section-dark py-section">
        <div className="container-main max-w-4xl">
          {/* No autoplay, no loop: the visitor chooses to play, native
              controls work without any JavaScript. */}
          <video
            controls
            playsInline
            preload="none"
            width={1280}
            height={720}
            poster="/assets/chenacolo_31.jpeg"
            className="w-full rounded-lg"
          >
            <source src="/assets/chenacolo_vid.mp4" type="video/mp4" />
          </video>
        </div>
      </section>

      <section className="section-light py-section">
        <div className="container-main grid max-w-6xl gap-10 lg:grid-cols-2 lg:items-center">
          <div className="relative aspect-[3/2] w-full overflow-hidden rounded-lg">
            <Image
              src="/assets/chenacolo_2.jpeg"
              alt="Recibidor de Villa Chenacolo"
              fill
              sizes="(min-width: 1024px) 50vw, 100vw"
              className="object-cover"
            />
          </div>
          <div className="space-y-4">
            <h2 className="type-h2 font-medium">Exquisita arquitectura</h2>
            <p className="type-body text-muted-foreground">
              Nada en Villa Chenacolo está puesto al azar. Cada arco, cada
              linea, cada textura fue elegida para crear un entorno funcional
              y estéticamente impecable.
            </p>
            <p className="type-body text-muted-foreground">
              Un lugar que no compite con tu evento: lo realza.
            </p>
            <p className="type-body text-muted-foreground">
              Más que una estructura, Villa Chenacolo es una experiencia
              arquitectónica que combina equilibrio, privacidad y distinción.
            </p>
            <p className="type-body text-muted-foreground">
              Todo está diseñado para que tus recuerdos encuentren el
              escenario perfecto.
            </p>
          </div>
        </div>
      </section>

      <section className="section-dark py-section">
        <div className="container-main max-w-6xl space-y-8">
          <h2 className="type-h2 text-center font-medium">Galería</h2>
          <ul className="grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-4">
            {galleryImages.map((src, index) => (
              <li key={src}>
                <a
                  href={src}
                  target="_blank"
                  rel="noopener noreferrer"
                  aria-label={`Ver imagen ${index + 1} del salón en tamaño completo`}
                  className="relative block aspect-square min-h-11 min-w-11 overflow-hidden rounded-lg transition-transform duration-300 hover:-translate-y-0.5 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-accent"
                >
                  <Image
                    src={src}
                    alt="Imagen del salón de eventos Villa Chenacolo"
                    fill
                    sizes="(min-width: 1024px) 25vw, (min-width: 640px) 33vw, 50vw"
                    className="object-cover"
                  />
                </a>
              </li>
            ))}
          </ul>
        </div>
      </section>
    </article>
  );
}
