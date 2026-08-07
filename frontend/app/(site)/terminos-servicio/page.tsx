import type { Metadata } from "next";

import { LegalPage } from "@/components/legal/legal-page";

// Copy migrated verbatim from
// internal/templates/pages/terminos-servicio.templ.
export const metadata: Metadata = {
  title: "Términos de Servicio",
  description:
    "Condiciones que rigen el uso de los servicios de Villa Chenacolo.",
};

export default function TermsOfServicePage() {
  return (
    <LegalPage
      title="Términos de Servicio"
      intro="Conoce las condiciones que rigen el uso de nuestros servicios en Villa Chenacolo. Transparencia y claridad en cada evento que celebramos contigo."
      lastUpdated="Agosto 2025"
      sections={[
        {
          heading: "Aceptación de Términos",
          body: (
            <>
              <p>
                Al contratar nuestros servicios o utilizar nuestras
                instalaciones, aceptas automáticamente estos términos y
                condiciones. Si no estás de acuerdo con alguna parte, te
                recomendamos no proceder con la contratación.
              </p>
              <p>
                Estos términos se aplican a todos los servicios ofrecidos por
                Villa Chenacolo, incluyendo:
              </p>
              <ul className="list-inside list-disc space-y-1">
                <li>Renta de salón principal y espacios exteriores</li>
                <li>Uso de capilla para ceremonias religiosas</li>
                <li>Servicios de coordinación y asistencia durante eventos</li>
                <li>
                  Acceso a instalaciones complementarias (cocina,
                  estacionamiento, etc.)
                </li>
              </ul>
            </>
          ),
        },
        {
          heading: "Reservaciones y Pagos",
          body: (
            <>
              <p>
                Para garantizar la disponibilidad de nuestras instalaciones,
                se requiere:
              </p>
              <ul className="list-inside list-disc space-y-1">
                <li>
                  <strong>Apartado:</strong> 50% del costo total para reservar
                  la fecha
                </li>
                <li>
                  <strong>Liquidación:</strong> El saldo restante debe pagarse
                  15 días antes del evento
                </li>
                <li>
                  <strong>Formas de pago:</strong> Efectivo, transferencia
                  bancaria o cheque certificado
                </li>
                <li>
                  <strong>Cotización válida:</strong> Los precios tienen
                  vigencia de 30 días naturales
                </li>
              </ul>
              <p>
                <strong>Importante:</strong> Las reservaciones se confirman
                únicamente mediante el pago del apartado correspondiente y la
                firma del contrato de servicios.
              </p>
            </>
          ),
        },
        {
          heading: "Política de Cancelaciones",
          body: (
            <>
              <p>
                Entendemos que pueden surgir circunstancias imprevistas.
                Nuestra política de cancelaciones es:
              </p>
              <ul className="list-inside list-disc space-y-1">
                <li>
                  <strong>Más de 60 días:</strong> Reembolso del 80% del
                  apartado
                </li>
                <li>
                  <strong>Entre 30-60 días:</strong> Reembolso del 50% del
                  apartado
                </li>
                <li>
                  <strong>Entre 15-30 días:</strong> Reembolso del 25% del
                  apartado
                </li>
                <li>
                  <strong>Menos de 15 días:</strong> No hay reembolso del
                  apartado
                </li>
              </ul>
              <p>
                En caso de fuerza mayor (desastres naturales, pandemias,
                etc.), evaluaremos cada caso individualmente para encontrar
                la mejor solución para ambas partes.
              </p>
            </>
          ),
        },
        {
          heading: "Responsabilidades del Cliente",
          body: (
            <>
              <p>Como cliente, te comprometes a:</p>
              <ul className="list-inside list-disc space-y-1">
                <li>Utilizar las instalaciones de manera responsable y respetuosa</li>
                <li>Cumplir con el horario acordado para el evento</li>
                <li>
                  Informar sobre el número exacto de invitados con 7 días de
                  anticipación
                </li>
                <li>Contratar proveedores autorizados y con experiencia comprobable</li>
                <li>Respetar las normas de seguridad y convivencia del lugar</li>
                <li>Hacerte responsable por cualquier daño causado por tus invitados</li>
              </ul>
              <p>
                Villa Chenacolo se reserva el derecho de suspender cualquier
                evento que no cumpla con estas condiciones.
              </p>
            </>
          ),
        },
        {
          heading: "Uso de Instalaciones",
          body: (
            <>
              <p>Nuestras instalaciones incluyen:</p>
              <ul className="list-inside list-disc space-y-1">
                <li>Salón principal climatizado con capacidad variable</li>
                <li>Capilla para ceremonias religiosas</li>
                <li>Jardines y espacios exteriores</li>
                <li>Cocina completamente equipada</li>
                <li>Estacionamiento amplio</li>
                <li>Baños con áreas privadas</li>
              </ul>
              <p>
                <strong>Restricciones:</strong> No se permite el uso de
                pirotecnia, música a volumen excesivo después de las 12:00
                AM, o decoraciones que dañen las instalaciones.
              </p>
            </>
          ),
        },
        {
          heading: "Limitaciones de Responsabilidad",
          body: (
            <>
              <p>Villa Chenacolo no se hace responsable por:</p>
              <ul className="list-inside list-disc space-y-1">
                <li>Objetos personales olvidados o extraviados durante el evento</li>
                <li>
                  Daños causados por proveedores externos contratados por el
                  cliente
                </li>
                <li>Interrupciones por fenómenos meteorológicos extremos</li>
                <li>
                  Fallas en servicios públicos (luz, agua) fuera de nuestro
                  control
                </li>
                <li>Accidentes causados por el consumo irresponsable de alcohol</li>
              </ul>
              <p>
                Recomendamos ampliamente contratar un seguro de evento para
                mayor tranquilidad.
              </p>
            </>
          ),
        },
        {
          heading: "Modificaciones y Contacto",
          body: (
            <>
              <p>
                Villa Chenacolo se reserva el derecho de modificar estos
                términos cuando sea necesario. Los cambios serán comunicados
                con anticipación razonable.
              </p>
              <p>
                Para consultas sobre estos términos o cualquier aspecto de
                nuestros servicios:
              </p>
              <p>
                <strong>Teléfono:</strong> +52 (618) 155-6407
                <br />
                <strong>Ubicación:</strong> Villa Chenacolo, Durango
              </p>
            </>
          ),
        },
      ]}
    />
  );
}
