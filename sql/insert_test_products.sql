-- Insertar categorías para materiales de organización de eventos
INSERT INTO categories (id, name, description, slug) VALUES 
('cc1962a3-739a-44ce-af8f-120db11ae7c9', 'Decoración', 'Elementos decorativos para fiestas, eventos corporativos y celebraciones', 'decoracion'),
('27f9205f-9aaa-4313-a2f6-f64d3e5b27ae', 'Mobiliario', 'Mesas, sillas y mobiliario para eventos y celebraciones', 'mobiliario'),
('2da80475-09ec-478a-a75b-286c0f188926', 'Vajilla y Cristalería', 'Platos, vasos, copas y cubiertos para eventos', 'vajilla-cristaleria'),
('79798154-2498-4847-b1e8-821db6a05afb', 'Iluminación', 'Sistemas de iluminación, velas y elementos luminosos', 'iluminacion'),
('0b2c667c-824b-4422-85c9-94a5914c9e8c', 'Sonido y Entretenimiento', 'Equipos de sonido, micrófonos y entretenimiento', 'sonido-entretenimiento'),
('082102ce-1088-4ac2-a517-f78ff03fa1f6', 'Textiles y Lencería', 'Manteles, servilletas y textiles para eventos', 'textiles-lenceria');

-- Insertar productos de decoración
INSERT INTO products (name, slug, description, long_description, category, available) VALUES 
('Globos Metálicos Dorados', 'globos-metalicos-dorados', 'Set de 20 globos metálicos dorados para fiestas', 'Globos metálicos de alta calidad en color dorado, perfectos para bodas, cumpleaños y eventos corporativos. Incluye 20 unidades.', 'cc1962a3-739a-44ce-af8f-120db11ae7c9', true),
('Centro de Mesa Floral', 'centro-mesa-floral', 'Arreglo floral elegante para centros de mesa', 'Hermoso centro de mesa con flores frescas de temporada, ideal para bodas, bautizos y eventos especiales. Altura aproximada 30cm.', 'cc1962a3-739a-44ce-af8f-120db11ae7c9', true),
('Guirnalda de Papel Tissue', 'guirnalda-papel-tissue', 'Guirnalda decorativa de papel tissue multicolor', 'Guirnalda de papel tissue de 3 metros de largo en colores pastel, perfecta para baby showers, bautizos y fiestas infantiles.', 'cc1962a3-739a-44ce-af8f-120db11ae7c9', true),
('Letras Luminosas LED', 'letras-luminosas-led', 'Letras LED personalizables para eventos', 'Letras luminosas LED de 40cm de altura, personalizables para nombres o palabras especiales. Ideales para bodas y eventos corporativos.', 'cc1962a3-739a-44ce-af8f-120db11ae7c9', true),
('Photobooth Props Set', 'photobooth-props-set', 'Set de accesorios para photobooth', 'Colección de 30 accesorios divertidos para photobooth: bigotes, gafas, sombreros y carteles. Perfecto para animar cualquier celebración.', 'cc1962a3-739a-44ce-af8f-120db11ae7c9', true);

-- Insertar productos de mobiliario
INSERT INTO products (name, slug, description, long_description, category, available) VALUES 
('Mesa Redonda para 8 Personas', 'mesa-redonda-8-personas', 'Mesa redonda de 1.5m para eventos', 'Mesa redonda de madera de 1.5 metros de diámetro, capacidad para 8 personas. Ideal para bodas, cenas de gala y eventos corporativos.', '27f9205f-9aaa-4313-a2f6-f64d3e5b27ae', true),
('Sillas Chiavari Doradas', 'sillas-chiavari-doradas', 'Sillas elegantes estilo Chiavari en dorado', 'Sillas Chiavari de resina en color dorado con cojín beige. Elegantes y cómodas, perfectas para bodas y eventos de lujo.', '27f9205f-9aaa-4313-a2f6-f64d3e5b27ae', true),
('Barra de Bar Portátil', 'barra-bar-portatil', 'Barra de bar móvil para eventos', 'Barra de bar portátil de 2 metros con ruedas y almacenamiento interno. Perfecta para cócteles, bodas y eventos corporativos.', '27f9205f-9aaa-4313-a2f6-f64d3e5b27ae', true),
('Podium con Micrófono', 'podium-con-microfono', 'Podium profesional para presentaciones', 'Podium de madera con sistema de micrófono integrado. Ideal para conferencias, eventos corporativos y ceremonias oficiales.', '27f9205f-9aaa-4313-a2f6-f64d3e5b27ae', true),
('Lounge Set Vintage', 'lounge-set-vintage', 'Set de muebles estilo vintage para lounge', 'Conjunto de sofás y mesas auxiliares estilo vintage. Incluye 2 sofás de 2 plazas y 2 mesas de centro. Perfecto para áreas de descanso.', '27f9205f-9aaa-4313-a2f6-f64d3e5b27ae', true);

-- Insertar productos de vajilla y cristalería
INSERT INTO products (name, slug, description, long_description, category, available) VALUES 
('Platos de Porcelana Blanca', 'platos-porcelana-blanca', 'Set de platos de porcelana para 50 personas', 'Juego completo de platos de porcelana blanca para 50 personas. Incluye platos llanos, hondos y de postre. Elegantes y resistentes.', '2da80475-09ec-478a-a75b-286c0f188926', true),
('Copas de Cristal para Vino', 'copas-cristal-vino', 'Copas de cristal para vino tinto y blanco', 'Set de 24 copas de cristal de alta calidad, 12 para vino tinto y 12 para vino blanco. Perfectas para bodas y cenas elegantes.', '2da80475-09ec-478a-a75b-286c0f188926', true),
('Cubiertos Dorados Premium', 'cubiertos-dorados-premium', 'Cubiertos de acero inoxidable con acabado dorado', 'Juego de cubiertos para 25 personas con acabado dorado. Incluye tenedor, cuchillo, cuchara y cucharilla de postre por persona.', '2da80475-09ec-478a-a75b-286c0f188926', true),
('Bandejas de Servicio Plateadas', 'bandejas-servicio-plateadas', 'Bandejas de servicio en acero inoxidable', 'Set de 6 bandejas de diferentes tamaños en acero inoxidable con acabado espejo. Ideales para servicio de catering y buffets.', '2da80475-09ec-478a-a75b-286c0f188926', true),
('Jarras de Cristal para Agua', 'jarras-cristal-agua', 'Jarras de cristal transparente para bebidas', 'Set de 8 jarras de cristal de 1.5 litros cada una. Perfectas para servir agua, jugos y bebidas en mesas de eventos.', '2da80475-09ec-478a-a75b-286c0f188926', true);

-- Insertar productos de iluminación
INSERT INTO products (name, slug, description, long_description, category, available) VALUES 
('Luces LED de Cadena', 'luces-led-cadena', 'Cadena de luces LED blancas cálidas', 'Cadena de 100 luces LED blancas cálidas de 10 metros de largo. Resistente al agua, perfecta para decoración exterior e interior.', '79798154-2498-4847-b1e8-821db6a05afb', true),
('Velas Flotantes Aromáticas', 'velas-flotantes-aromaticas', 'Velas flotantes con aroma a vainilla', 'Set de 20 velas flotantes aromáticas con fragancia a vainilla. Duración de 4 horas cada una. Ideales para centros de mesa con agua.', '79798154-2498-4847-b1e8-821db6a05afb', true),
('Proyector de Luces Disco', 'proyector-luces-disco', 'Proyector LED con efectos de luces de colores', 'Proyector LED con múltiples efectos de luces de colores y patrones. Control remoto incluido. Perfecto para fiestas y eventos juveniles.', '79798154-2498-4847-b1e8-821db6a05afb', true),
('Candelabros de Mesa Dorados', 'candelabros-mesa-dorados', 'Candelabros elegantes de 5 brazos', 'Set de 4 candelabros dorados de 5 brazos cada uno. Altura de 40cm. Incluye velas blancas. Perfectos para cenas elegantes y bodas.', '79798154-2498-4847-b1e8-821db6a05afb', true),
('Faroles de Papel Japonés', 'faroles-papel-japones', 'Faroles de papel en colores pastel', 'Set de 12 faroles de papel japonés en colores pastel de diferentes tamaños. Incluye sistema de iluminación LED. Ideales para exteriores.', '79798154-2498-4847-b1e8-821db6a05afb', true);

-- Insertar productos de sonido y entretenimiento
INSERT INTO products (name, slug, description, long_description, category, available) VALUES 
('Sistema de Sonido Portátil', 'sistema-sonido-portatil', 'Equipo de sonido con micrófonos inalámbricos', 'Sistema de sonido completo con altavoces de 500W, mezcladora y 2 micrófonos inalámbricos. Perfecto para eventos de hasta 200 personas.', '0b2c667c-824b-4422-85c9-94a5914c9e8c', true),
('Karaoke Profesional', 'karaoke-profesional', 'Sistema de karaoke con pantalla y canciones', 'Sistema de karaoke profesional con pantalla de 32", base de datos de 5000 canciones y 4 micrófonos. Ideal para fiestas y eventos familiares.', '0b2c667c-824b-4422-85c9-94a5914c9e8c', true),
('DJ Booth Completo', 'dj-booth-completo', 'Cabina de DJ con mesa y equipos', 'Cabina de DJ completa con mesa, controlador, altavoces y sistema de luces. Incluye biblioteca musical de diferentes géneros.', '0b2c667c-824b-4422-85c9-94a5914c9e8c', true),
('Pantalla de Proyección', 'pantalla-proyeccion', 'Pantalla portátil para presentaciones', 'Pantalla de proyección portátil de 3x2 metros con trípode. Incluye proyector HD. Perfecta para presentaciones corporativas y eventos.', '082102ce-1088-4ac2-a517-f78ff03fa1f6', true),
('Animación Infantil Set', 'animacion-infantil-set', 'Kit completo para animación de fiestas infantiles', 'Kit de animación que incluye juegos, globoflexia, pintacaritas y música infantil. Perfecto para cumpleaños y eventos familiares.', '0b2c667c-824b-4422-85c9-94a5914c9e8c', true);

-- Insertar productos de textiles y lencería
INSERT INTO products (name, slug, description, long_description, category, available) VALUES 
('Manteles de Lino Blanco', 'manteles-lino-blanco', 'Manteles de lino para mesas redondas y rectangulares', 'Set de manteles de lino 100% natural en color blanco. Incluye 10 manteles redondos y 10 rectangulares de diferentes tamaños.', '082102ce-1088-4ac2-a517-f78ff03fa1f6', true),
('Servilletas de Tela Bordadas', 'servilletas-tela-bordadas', 'Servilletas de tela con bordado elegante', 'Set de 100 servilletas de tela de algodón con bordado en hilo dorado. Disponibles en colores blanco, marfil y champagne.', '082102ce-1088-4ac2-a517-f78ff03fa1f6', true),
('Caminos de Mesa Dorados', 'caminos-mesa-dorados', 'Caminos de mesa con detalles dorados', 'Set de 15 caminos de mesa de 2.5 metros con detalles bordados en hilo dorado. Perfectos para mesas largas en bodas y eventos elegantes.', '082102ce-1088-4ac2-a517-f78ff03fa1f6', true),
('Fundas para Sillas Universales', 'fundas-sillas-universales', 'Fundas elásticas blancas para sillas', 'Set de 50 fundas universales elásticas en color blanco. Se adaptan a la mayoría de sillas. Incluye lazos dorados para decoración.', '082102ce-1088-4ac2-a517-f78ff03fa1f6', true),
('Cortinas de Fondo para Eventos', 'cortinas-fondo-eventos', 'Cortinas decorativas para fondos de ceremonia', 'Set de cortinas de gasa blanca de 4x3 metros con sistema de instalación. Perfectas para crear fondos elegantes en ceremonias y fotos.', '082102ce-1088-4ac2-a517-f78ff03fa1f6', true);
