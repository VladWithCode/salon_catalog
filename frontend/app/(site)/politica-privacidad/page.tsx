import type { Metadata } from "next";

import { LegalPage } from "@/components/legal/legal-page";

// Copy migrated verbatim from
// internal/templates/pages/politica-privacidad.templ.
export const metadata: Metadata = {
  title: "Política de Privacidad",
  description:
    "Conoce cómo Villa Chenacolo protege y utiliza tu información personal.",
};

export default function PrivacyPolicyPage() {
  return (
    <LegalPage
      title="Política de Privacidad"
      intro="En Villa Chenacolo valoramos y respetamos tu privacidad. Conoce cómo protegemos y utilizamos tu información personal para brindarte la mejor experiencia en nuestros servicios."
      lastUpdated="Agosto 2025"
      sections={[
        {
          heading: "Información que Recolectamos",
          body: (
            <>
              <p>
                Recolectamos información que nos proporcionas directamente
                cuando utilizas nuestros servicios, incluyendo:
              </p>
              <ul className="list-inside list-disc space-y-1">
                <li>Nombre completo y datos de contacto (teléfono, email)</li>
                <li>Información sobre eventos y fechas de celebración</li>
                <li>Preferencias de servicios y requerimientos especiales</li>
                <li>Historial de comunicaciones y consultas</li>
              </ul>
            </>
          ),
        },
        {
          heading: "Cómo Utilizamos tu Información",
          body: (
            <>
              <p>Utilizamos la información recolectada exclusivamente para:</p>
              <ul className="list-inside list-disc space-y-1">
                <li>Coordinar y confirmar reservaciones de eventos</li>
                <li>Personalizar nuestros servicios según tus necesidades</li>
                <li>Comunicarnos contigo sobre tu evento o consultas</li>
                <li>Mejorar continuamente nuestros servicios</li>
                <li>
                  Enviar información relevante sobre promociones (solo con tu
                  consentimiento)
                </li>
              </ul>
            </>
          ),
        },
        {
          heading: "Protección de tu Información",
          body: (
            <>
              <p>
                Implementamos medidas de seguridad técnicas y administrativas
                apropiadas para proteger tu información personal:
              </p>
              <ul className="list-inside list-disc space-y-1">
                <li>Almacenamiento seguro de datos con acceso limitado</li>
                <li>Cifrado de información sensible durante la transmisión</li>
                <li>Capacitación regular de nuestro personal sobre privacidad</li>
                <li>Revisión periódica de nuestros procedimientos de seguridad</li>
              </ul>
              <p>
                <strong>Importante:</strong> No vendemos, alquilamos ni
                compartimos tu información personal con terceros para fines
                comerciales sin tu consentimiento expreso.
              </p>
            </>
          ),
        },
        {
          heading: "Tus Derechos",
          body: (
            <>
              <p>Tienes derecho a:</p>
              <ul className="list-inside list-disc space-y-1">
                <li>Acceder a la información personal que tenemos sobre ti</li>
                <li>Solicitar correcciones o actualizaciones de tus datos</li>
                <li>Solicitar la eliminación de tu información personal</li>
                <li>Retirar tu consentimiento para el procesamiento de datos</li>
                <li>Recibir una copia de tu información en formato portable</li>
              </ul>
              <p>
                Para ejercer cualquiera de estos derechos, contáctanos
                directamente a través de nuestros canales oficiales.
              </p>
            </>
          ),
        },
        {
          heading: "Cookies y Tecnologías Similares",
          body: (
            <>
              <p>Nuestro sitio web utiliza cookies esenciales para:</p>
              <ul className="list-inside list-disc space-y-1">
                <li>Mantener tus preferencias de navegación</li>
                <li>Recordar items en tu carrito de cotización</li>
                <li>Mejorar la funcionalidad del sitio web</li>
                <li>Proporcionar una experiencia personalizada</li>
              </ul>
              <p>
                Puedes controlar las cookies a través de la configuración de
                tu navegador, aunque esto podría afectar la funcionalidad del
                sitio.
              </p>
            </>
          ),
        },
        {
          heading: "Contacto y Actualizaciones",
          body: (
            <>
              <p>
                Si tienes preguntas sobre esta Política de Privacidad o sobre
                cómo manejamos tu información personal, no dudes en
                contactarnos:
              </p>
              <p>
                <strong>Teléfono:</strong> 618-259-3026
                <br />
                <strong>Ubicación:</strong> Villa Chenacolo, Durango
              </p>
              <p>
                Esta política puede ser actualizada periódicamente. Te
                notificaremos sobre cambios importantes a través de nuestros
                canales oficiales.
              </p>
            </>
          ),
        },
      ]}
    />
  );
}
