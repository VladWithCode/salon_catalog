export const eventKeys = [
  "bodas",
  "quinceaneras",
  "bautizos",
  "corporativos",
  "graduaciones",
  "privadas",
] as const;

export type EventKey = (typeof eventKeys)[number];

export type EventBaseCopy = Readonly<{
  key: EventKey;
  canonicalName: string;
  description: string;
  highlights: readonly string[];
  footnote?: string;
  whatsappMessage: string;
}>;

export const eventsByKey = {
  bodas: {
    key: "bodas",
    canonicalName: "Bodas",
    description:
      "El ‘sí’ más importante merece un lugar como este. Capilla íntima y elegante, salón cerrado con aire acondicionado, acústica impecable y áreas exteriores que se integran con armonía.",
    highlights: [
      "Renta de salón y capilla",
      "Espacio con capacidad ampliable (hasta 200 personas más al abrirse)",
      "Climatización total",
      "Iluminación y acústica premium",
      "Cocina completamente equipada para tu proveedor de banquete",
      "Estacionamiento amplio dentro y fuera de la villa",
      "Personal de asistencia, valet y seguridad",
    ],
    footnote: "El mobiliario disponible se renta por separado.",
    whatsappMessage:
      "¡Hola! Me gustaría conocer más sobre la planeación de bodas de Salón Chenacolo.",
  },
  quinceaneras: {
    key: "quinceaneras",
    canonicalName: "Quinceañeras y celebraciones",
    description:
      "Diseña la fiesta de tus sueños en un espacio que se transforma contigo. Desde temáticas juveniles hasta cenas formales, el salón se adapta a tu visión.",
    highlights: [
      "Salón climatizado y cerrado para eventos íntimos o fiestas grandes",
      "Fuente, puente, jardín y área techada al aire libre para tematizar",
      "Cocina equipada lista para tu proveedor de catering",
      "Distribución funcional para áreas de foto, comida y pista de baile",
      "Personal en baños, seguridad y valet",
    ],
    whatsappMessage:
      "¡Hola! Me gustaría conocer más sobre las quinceañeras de Salón Chenacolo.",
  },
  bautizos: {
    key: "bautizos",
    canonicalName: "Bautizos y comuniones",
    description:
      "Celebraciones con alma y buen gusto. La capilla es perfecta para ceremonias familiares, seguidas de un evento en el salón o el jardín.",
    highlights: [
      "Renta de capilla y salón (opcional por separado)",
      "Espacios ideales para fotos y reuniones familiares",
      "Cocina equipada para preparar o calentar alimentos",
      "Baños impecables con sala privada en el área de mujeres",
      "Atención personalizada y servicio discreto",
    ],
    whatsappMessage:
      "¡Hola! Me gustaría conocer más sobre los bautizos de Salón Chenacolo.",
  },
  corporativos: {
    key: "corporativos",
    canonicalName: "Eventos corporativos",
    description:
      "Un entorno elegante para eventos profesionales. Ya sea para una cena empresarial, una entrega de premios o un lanzamiento de producto, Chenacolo proyecta profesionalismo y alto nivel.",
    highlights: [
      "Salón con capacidad para grandes montajes",
      "Estacionamiento para equipo e invitados",
      "Apoyo en logística de acceso y seguridad",
      "Zona exterior techada para actividades complementarias",
      "Cocina y baños disponibles para todo el personal externo",
    ],
    whatsappMessage:
      "¡Hola! Me gustaría conocer más sobre los eventos corporativos de Salón Chenacolo.",
  },
  graduaciones: {
    key: "graduaciones",
    canonicalName: "Graduaciones",
    description:
      "Celebra logros en un lugar con estilo y amplitud. Un espacio seguro, bien distribuido y con todo lo necesario para una noche inolvidable.",
    highlights: [
      "Salón principal climatizado",
      "Zonas exteriores para fotos y áreas de espera",
      "Cocina lista para uso profesional",
      "Iluminación y acústica para ceremonia o fiesta",
      "Servicio continuo de limpieza en baños y apoyo logístico",
    ],
    whatsappMessage:
      "¡Hola! Me gustaría conocer más sobre las graduaciones de Salón Chenacolo.",
  },
  privadas: {
    key: "privadas",
    canonicalName: "Fiestas privadas",
    description:
      "De lo íntimo a lo espectacular. Celebra cumpleaños, aniversarios o reuniones familiares en un entorno que se siente privado, exclusivo y cuidado.",
    highlights: [
      "Espacios personalizables: salón, jardín y zona techada trasera",
      "Baños amplios y en constante mantenimiento",
      "Distribución arquitectónica que permite que el evento fluya sin interrupciones",
      "Privacidad gracias a la ubicación",
      "Personal discreto y atento en todo momento",
    ],
    whatsappMessage:
      "¡Hola! Me gustaría conocer más sobre las fiestas privadas de Salón Chenacolo.",
  },
} as const satisfies Readonly<Record<EventKey, EventBaseCopy>>;
