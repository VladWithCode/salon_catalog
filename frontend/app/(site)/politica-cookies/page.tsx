import type { Metadata } from "next";

import { LegalPage } from "@/components/legal/legal-page";

// Copy migrated verbatim from
// internal/templates/pages/politica-cookies.templ.
export const metadata: Metadata = {
  title: "Política de Cookies",
  description: "Cómo Villa Chenacolo utiliza cookies en su sitio web.",
};

export default function CookiePolicyPage() {
  return (
    <LegalPage
      title="Política de Cookies"
      intro="Información detallada sobre el uso de cookies en nuestro sitio web. Transparencia total sobre cómo mejoramos tu experiencia de navegación."
      lastUpdated="Agosto 2025"
      sections={[
        {
          heading: "¿Qué son las Cookies?",
          body: (
            <>
              <p>
                Las cookies son pequeños archivos de texto que se almacenan en
                tu dispositivo cuando visitas nuestro sitio web. Estas nos
                ayudan a recordar tus preferencias y mejorar tu experiencia de
                navegación.
              </p>
              <p>
                Villa Chenacolo utiliza cookies de manera responsable y
                transparente, siempre respetando tu privacidad y dándote
                control sobre su uso.
              </p>
              <p>
                <strong>Nota importante:</strong> Nuestro sitio web puede
                funcionar sin cookies, pero algunas funcionalidades podrían
                verse limitadas.
              </p>
            </>
          ),
        },
        {
          heading: "Tipos de Cookies que Utilizamos",
          body: (
            <>
              <p>
                <strong>Cookies técnicas (esenciales):</strong> imprescindibles
                para el funcionamiento básico del sitio web.
              </p>
              <ul className="list-inside list-disc space-y-1">
                <li>Mantener tu carrito de cotización activo</li>
                <li>Recordar tu sesión de usuario si inicias sesión</li>
                <li>Garantizar la seguridad durante la navegación</li>
              </ul>
              <p>
                <strong>Cookies de funcionalidad:</strong> mejoran tu
                experiencia recordando tus preferencias.
              </p>
              <ul className="list-inside list-disc space-y-1">
                <li>Configuración de idioma o región</li>
                <li>Preferencias de visualización de galería</li>
                <li>Filtros aplicados en el catálogo</li>
              </ul>
            </>
          ),
        },
        {
          heading: "Duración y Almacenamiento",
          body: (
            <>
              <p>
                Las cookies que utilizamos tienen diferentes períodos de vida
                según su función:
              </p>
              <p>
                <strong>Cookies de sesión:</strong> se eliminan
                automáticamente al cerrar tu navegador. Se usan para
                funciones básicas como mantener tu carrito activo.
              </p>
              <p>
                <strong>Cookies persistentes:</strong> permanecen hasta 30
                días para recordar preferencias como filtros del catálogo y
                configuraciones de usuario.
              </p>
              <p>
                <strong>Importante:</strong> Todas las cookies se almacenan
                únicamente en tu dispositivo y nunca contienen información
                personal identificable sin tu consentimiento.
              </p>
            </>
          ),
        },
        {
          heading: "Control y Gestión de Cookies",
          body: (
            <>
              <p>Tienes control total sobre las cookies. Puedes:</p>
              <ul className="list-inside list-disc space-y-1">
                <li>Aceptar o rechazar cookies no esenciales</li>
                <li>Modificar la configuración de cookies en cualquier momento</li>
                <li>
                  Eliminar cookies existentes desde la configuración de tu
                  navegador
                </li>
                <li>
                  Navegar en modo privado/incógnito para evitar el
                  almacenamiento
                </li>
              </ul>
              <p>
                Todos los navegadores modernos permiten controlar cookies
                desde sus configuraciones de privacidad. Consulta la ayuda de
                tu navegador específico para instrucciones detalladas.
              </p>
            </>
          ),
        },
        {
          heading: "Cookies de Terceros",
          body: (
            <>
              <p>
                En Villa Chenacolo no utilizamos cookies de terceros para
                publicidad o seguimiento. Sin embargo, algunos servicios
                integrados pueden usar sus propias cookies:
              </p>
              <p>
                <strong>Mapas y ubicación:</strong> si integramos mapas de
                ubicación, estos servicios pueden usar cookies para mejorar la
                funcionalidad del mapa.
              </p>
              <p>
                <strong>Sistemas de chat:</strong> los widgets de chat o
                WhatsApp Web pueden usar cookies para mantener conversaciones
                activas.
              </p>
              <p>
                <strong>Compromiso:</strong> Siempre te informaremos si
                introducimos nuevos servicios de terceros que utilicen
                cookies.
              </p>
            </>
          ),
        },
        {
          heading: "Actualizaciones y Contacto",
          body: (
            <>
              <p>
                Esta política de cookies puede actualizarse para reflejar
                cambios en nuestro sitio web o en la legislación aplicable.
                Te notificaremos de cambios significativos.
              </p>
              <p>
                Si tienes preguntas sobre nuestro uso de cookies o deseas
                ejercer tus derechos:
              </p>
              <p>
                <strong>Teléfono:</strong> +52 (618) 155-6407
                <br />
                <strong>Ubicación:</strong> Villa Chenacolo, Durango
              </p>
              <p>
                <strong>Tu privacidad es importante para nosotros.</strong>{" "}
                Utilizamos cookies únicamente para mejorar tu experiencia y
                nunca para comprometer tu privacidad.
              </p>
            </>
          ),
        },
      ]}
    />
  );
}
