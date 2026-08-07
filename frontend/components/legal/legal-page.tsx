import type { ReactNode } from "react";

export type LegalSection = Readonly<{
  heading: string;
  body: ReactNode;
}>;

type LegalPageProps = Readonly<{
  title: string;
  intro: string;
  sections: readonly LegalSection[];
  lastUpdated: string;
}>;

/**
 * Shared shell for the three legal pages (privacy, terms, cookies). All
 * copy is passed in by each page verbatim from the Go templates it
 * replaces (internal/templates/pages/politica-privacidad.templ,
 * terminos-servicio.templ, politica-cookies.templ) — nothing here
 * generates or completes legal text.
 */
export function LegalPage({ title, intro, sections, lastUpdated }: LegalPageProps) {
  return (
    <article className="section-light py-section">
      <div className="container-main max-w-3xl space-y-section">
        <div className="space-y-4">
          <h1 className="type-h1 font-medium">{title}</h1>
          <p className="type-lead text-muted-foreground">{intro}</p>
        </div>

        <div className="space-y-10">
          {sections.map((section) => (
            <section key={section.heading} className="space-y-3">
              <h2 className="type-h3 font-medium">{section.heading}</h2>
              <div className="type-body measure-body space-y-3 text-muted-foreground">
                {section.body}
              </div>
            </section>
          ))}
        </div>

        <p className="type-small text-muted-foreground/70">
          Última actualización: {lastUpdated}
        </p>
      </div>
    </article>
  );
}
