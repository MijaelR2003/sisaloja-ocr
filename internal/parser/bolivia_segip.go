package parser

import (
	"regexp"
	"strings"
	"unicode"
)

// BoliviaDocumentAnalysis describe el diagnóstico especializado de un documento boliviano (SEGIP).
type BoliviaDocumentAnalysis struct {
	IsBolivia        bool           `json:"is_bolivia"`
	Subtype          string         `json:"subtype"`           // SEGIP_NUEVO | SEGIP_CLASICO | SEGIP_MRZ_IDBOL | PASAPORTE_BOL
	ExtractionMethod string         `json:"extraction_method"` // MRZ_IDBOL | ETIQUETAS_SEGIP | VENTANA_ESTRUCTURAL_BOLIVIA | TITULAR_CLASICO
	ConfidenceScore  int            `json:"confidence_score"`  // 0 - 100
	Fields           DocumentFields `json:"fields"`
}

// Base léxica incrustada de apellidos frecuentes en Bolivia (Altiplano, Valles y Oriente)
var boliviaSurnamesSet = map[string]bool{
	"MAMANI": true, "QUISPE": true, "CONDORI": true, "CHOQUE": true, "FLORES": true,
	"VARGAS": true, "ROJAS": true, "TORREZ": true, "TORRES": true, "SILVA": true,
	"GUTIERREZ": true, "GUTIÉRREZ": true, "MORALES": true, "PINTO": true, "HUANCA": true,
	"NINA": true, "APAZA": true, "TICONA": true, "CRUZ": true, "MENDOZA": true,
	"GONZALES": true, "GONZALEZ": true, "GONZÁLEZ": true, "RODRIGUEZ": true, "RODRÍGUEZ": true,
	"PEREZ": true, "PÉREZ": true, "LOPEZ": true, "LÓPEZ": true, "GARCIA": true, "GARCÍA": true,
	"SANCHEZ": true, "SÁNCHEZ": true, "FERNANDEZ": true, "FERNÁNDEZ": true, "CAMACHO": true,
	"JUSTINIANO": true, "SAUCEDO": true, "VACA": true, "CUELLAR": true, "CUÉLLAR": true,
	"CHAVEZ": true, "CHÁVEZ": true, "SUAREZ": true, "SUÁREZ": true, "RIVERO": true,
	"HURTADO": true, "ARTEAGA": true, "PAZ": true, "AGUILERA": true, "CASTELLON": true,
	"CASTELLÓN": true, "CLAROS": true, "MONTANO": true, "MONTAÑO": true, "ZURITA": true,
	"LEDEZMA": true, "VILLARROEL": true, "GUZMAN": true, "GUZMÁN": true, "TERCEROS": true,
	"BALDERRAMA": true, "VELASQUEZ": true, "VELÁSQUEZ": true, "AGUILAR": true, "MIRANDA": true,
	"ESPINOZA": true, "CALLISAYA": true, "CALIZAYA": true, "YUJRA": true, "LIMACHI": true,
	"POMA": true, "LAURA": true, "TARQUI": true, "COLQUE": true, "VILLCA": true,
	"COPA": true, "RAMOS": true, "CASTRO": true, "ROMERO": true, "ORTEGA": true,
	"SALAZAR": true, "CABRERA": true, "SOTO": true, "MEDINA": true, "DELGADO": true,
	"MOLINA": true, "ORTIZ": true, "TAPIA": true, "PAREDES": true, "MALDONADO": true,
	"CARRILLO": true, "PENA": true, "PEÑA": true, "RIOS": true, "RÍOS": true,
	"ACOSTA": true, "VEGA": true, "ARANCIBIA": true, "ARCE": true, "ARIAS": true,
	"AYALA": true, "BALDIVIEZO": true, "BARRIENTOS": true, "BEJARANO": true, "BLANCO": true,
	"BORDA": true, "BUSTAMANTE": true, "CABALLERO": true, "CACERES": true, "CÁCERES": true,
	"CALLE": true, "CALLEJAS": true, "CANAVIRI": true, "CARDENAS": true, "CÁRDENAS": true,
	"CARRASCO": true, "CARVAJAL": true, "CASTILLO": true, "CATACORA": true, "CEBALLOS": true,
	"CENTELLAS": true, "CESPEDES": true, "CÉSPEDES": true, "CHAMBI": true, "CHIPANA": true,
	"CHOQUEHUANCA": true, "CHURA": true, "CLAURE": true, "COCA": true, "COLODRO": true,
	"CORDERO": true, "CORONADO": true, "CORTEZ": true, "COSSIO": true, "CRESPO": true,
	"CUETO": true, "DAVALOS": true, "DÁVALOS": true, "DAZA": true, "DIAZ": true, "DÍAZ": true,
	"DORADO": true, "DURAN": true, "DURÁN": true, "ENCINAS": true, "ERGUETA": true,
	"ESCALANTE": true, "ESCOBAR": true, "ESPEJO": true, "ESTRADA": true, "FARFAN": true,
	"FARFÁN": true, "FERRUFINO": true, "FIGUEROA": true, "FRANCO": true, "FUENTES": true,
	"GALARZA": true, "GALLARDO": true, "GALVEZ": true, "GÁLVEZ": true, "GAMARRA": true,
	"GARRIDO": true, "GIL": true, "GOMEZ": true, "GÓMEZ": true, "GOYTIA": true,
	"GRAGEDA": true, "GUARDIA": true, "GUERRA": true, "GUERRERO": true, "GUEVARA": true,
	"GUMUCIO": true, "HERBAS": true, "HEREDIA": true, "HERRERA": true, "HIDALGO": true,
	"HINOJOSA": true, "HOYOS": true, "HUARACHI": true, "HUMEREZ": true, "IBANEZ": true,
	"IBAÑEZ": true, "IBÁÑEZ": true, "ILLANES": true, "IRIARTE": true, "ITURRI": true,
	"JAIMES": true, "JIMENEZ": true, "JIMÉNEZ": true, "JORDAN": true, "JORDÁN": true,
	"JUAREZ": true, "JUÁREZ": true, "LARA": true, "LAZARTE": true, "LAZO": true,
	"LEMA": true, "LEON": true, "LEÓN": true, "LLANOS": true, "LOAYZA": true,
	"LOBO": true, "LORA": true, "LOZADA": true, "LOZANO": true, "LUNA": true,
	"MACHACA": true, "MACIAS": true, "MANSILLA": true, "MARISCAL": true, "MARQUEZ": true,
	"MÁRQUEZ": true, "MARTINEZ": true, "MARTÍNEZ": true, "MAYDANA": true, "MEDRANO": true,
	"MEJIA": true, "MEJÍA": true, "MELGAR": true, "MENDEZ": true, "MÉNDEZ": true,
	"MERCADO": true, "MERIDA": true, "MÉRIDA": true, "MICHEL": true, "MIER": true,
	"MOJICA": true, "MONASTERIO": true, "MONROY": true, "MONTERO": true, "MONTES": true,
	"MONTESINOS": true, "MORENO": true, "MOSCOSO": true, "MOYA": true, "MUNOZ": true,
	"MUÑOZ": true, "MURILLO": true, "NAVARRO": true, "NAVIA": true, "NAVA": true,
	"NOGALES": true, "NUNEZ": true, "NUÑEZ": true, "NÚÑEZ": true, "OBLITAS": true,
	"OCAMPO": true, "OCHOA": true, "OLGUIN": true, "OLIVARES": true, "OLIVERA": true,
	"ORELLANA": true, "ORDONEZ": true, "ORDOÑEZ": true, "OROZCO": true, "OVANDO": true,
	"CHUMACERO": true, "PACHECO": true, "PADILLA": true, "PALACIOS": true, "PALENQUE": true,
	"PANIAGUA": true, "PANTOJA": true, "PARADA": true, "PARDO": true, "PEDRAZA": true,
	"PENARANDA": true, "PEÑARANDA": true, "PERALTA": true, "PEREIRA": true, "PEREDO": true,
	"PIMENTEL": true, "PINEDO": true, "POZO": true, "PRADO": true, "PRIETO": true,
	"PUMA": true, "QUEVEDO": true, "QUINTANA": true, "QUINTEROS": true, "QUIROGA": true,
	"QUISBERT": true, "RAMIREZ": true, "RAMÍREZ": true, "REVOLLO": true, "RIBERA": true,
	"RICO": true, "RIOJA": true, "RIVAS": true, "RIVERA": true, "ROCA": true,
	"ROCABADO": true, "ROCHA": true, "RODAS": true, "ROLDAN": true, "ROLDÁN": true,
	"ROMAN": true, "ROMÁN": true, "ROSALES": true, "ROSSEL": true, "RUIZ": true,
	"SAAVEDRA": true, "SAINZ": true, "SALAS": true, "SALCEDO": true, "SALGUERO": true,
	"SALINAS": true, "SANJINES": true, "SANDOVAL": true, "SANTANA": true, "SANTIVANEZ": true,
	"SANTIVAÑEZ": true, "SANTOS": true, "SARAVIA": true, "SARMIENTO": true, "SERRANO": true,
	"SILES": true, "SOLIZ": true, "SOLIS": true, "SORIA": true, "SORUCO": true,
	"TABOADA": true, "TELLEZ": true, "TÉLLEZ": true, "TERAN": true, "TERÁN": true,
	"TERRAZAS": true, "TEJADA": true, "TITO": true, "TOLEDO": true, "TOLA": true,
	"TORRICO": true, "TUDELA": true, "UGARTE": true, "ULLOA": true, "URIONA": true,
	"URQUIDI": true, "URQUIZO": true, "VALDA": true, "VALDEZ": true, "VALDIVIA": true,
	"VALENCIA": true, "VALENZUELA": true, "VALLE": true, "VALLEJOS": true, "VASQUEZ": true,
	"VÁSQUEZ": true, "VEDIA": true, "VELASCO": true, "VELARDE": true, "VELEZ": true,
	"VENEGAS": true, "VERA": true, "VERDUGUEZ": true, "VIDAURRE": true, "VIDEA": true,
	"VILLEGAS": true, "VIRUEZ": true, "VISCARRA": true, "YANEZ": true, "YAÑEZ": true,
	"ZAMBRANA": true, "ZAMORA": true, "ZAMORANO": true, "ZAPATA": true, "ZARATE": true,
	"ZÁRATE": true, "ZEBALLOS": true, "ZEGARRA": true, "ZELADA": true, "ZENTENO": true,
	"ZUAZO": true, "ZUNIGA": true, "ZUÑIGA": true, "ZÚÑIGA": true,
}

// Base léxica incrustada de nombres de pila frecuentes en Bolivia
var boliviaFirstNamesSet = map[string]bool{
	"MIJAEL": true, "ALEJANDRO": true, "JUAN": true, "CARLOS": true, "JOSE": true, "JOSÉ": true,
	"LUIS": true, "MIGUEL": true, "ANGEL": true, "ÁNGEL": true, "JORGE": true, "MARIO": true,
	"ROBERTO": true, "FERNANDO": true, "DANIEL": true, "DAVID": true, "GABRIEL": true,
	"CRISTIAN": true, "CHRISTIAN": true, "RODRIGO": true, "MARCELO": true, "MAURICIO": true,
	"GONZALO": true, "DIEGO": true, "PABLO": true, "PEDRO": true, "MARCO": true, "MARCOS": true,
	"ANTONIO": true, "JULIO": true, "CESAR": true, "CÉSAR": true, "VICTOR": true, "VÍCTOR": true,
	"HUGO": true, "OSCAR": true, "ÓSCAR": true, "EDGAR": true, "EDWIN": true, "EDDY": true,
	"FRANZ": true, "LIMBERT": true, "LIMBER": true, "GROVER": true, "WILDER": true, "WILMER": true,
	"WILLY": true, "WILSON": true, "MILTON": true, "NELSON": true, "RONALD": true, "RAMIRO": true,
	"RUBEN": true, "RUBÉN": true, "RAUL": true, "RAÚL": true, "SERGIO": true, "GUSTAVO": true,
	"ALVARO": true, "ÁLVARO": true, "ARIEL": true, "ADRIAN": true, "ADRIÁN": true, "ANDRES": true,
	"ANDRÉS": true, "ARTURO": true, "BERNARDO": true, "BORIS": true, "BRUNO": true, "CAMILO": true,
	"CARMELO": true, "CELSO": true, "CLAUDIO": true, "CLEMENTE": true, "DARIO": true, "DARÍO": true,
	"DENIS": true, "DENNIS": true, "EFRAIN": true, "EFRAÍN": true, "ELIAS": true, "ELÍAS": true,
	"ELOY": true, "EMILIO": true, "ENRIQUE": true, "ERICK": true, "ERIK": true, "ERNESTO": true,
	"ESTEBAN": true, "EUGENIO": true, "FABIAN": true, "FABIÁN": true, "FABRICIO": true,
	"FAUSTO": true, "FEDERICO": true, "FELIPE": true, "FELIX": true, "FÉLIX": true, "FIDEL": true,
	"FLAVIO": true, "FRANCISCO": true, "FRANKLIN": true, "FREDDY": true, "GERMAN": true,
	"GERMÁN": true, "GILBERTO": true, "GIOVANI": true, "GREGORIO": true, "GUIDO": true,
	"GUILLERMO": true, "HECTOR": true, "HÉCTOR": true, "HENRY": true, "HERNAN": true,
	"HERNÁN": true, "HILARION": true, "HORACIO": true, "HUMBERTO": true, "IGNACIO": true,
	"IVAN": true, "IVÁN": true, "JAIME": true, "JAVIER": true, "JESUS": true, "JESÚS": true,
	"JHON": true, "JHONNY": true, "JOHNNY": true, "JOAQUIN": true, "JOAQUÍN": true, "JOEL": true,
	"JONATHAN": true, "JOSUE": true, "JOSUÉ": true, "JULIAN": true, "JULIÁN": true, "JUNIOR": true,
	"JUSTO": true, "KEVIN": true, "LEONARDO": true, "LEONEL": true, "LORENZO": true, "LUCAS": true,
	"LUCIO": true, "MANUEL": true, "AGUSTIN": true, "AGUSTÍN": true, "MARTIN": true, "MARTÍN": true,
	"MATEO": true, "MATIAS": true, "MATÍAS": true, "MAX": true, "MAXIMO": true, "MÁXIMO": true,
	"MOISES": true, "MOISÉS": true, "NESTOR": true, "NÉSTOR": true, "NICOLAS": true, "NICOLÁS": true,
	"NOLBERTO": true, "NORBERTO": true, "OCTAVIO": true, "OLIVER": true, "OMAR": true,
	"ORLANDO": true, "OSVALDO": true, "OSWALDO": true, "PATRICIO": true, "PAUL": true, "PAÚL": true,
	"PERCY": true, "RAFAEL": true, "RENE": true, "RENÉ": true, "RICARDO": true, "RICHARD": true,
	"RODOLFO": true, "ROGER": true, "ROLANDO": true, "ROMULO": true, "RÓMULO": true,
	"RUFINO": true, "SAMUEL": true, "SANTIAGO": true, "SAUL": true, "SAÚL": true,
	"SEBASTIAN": true, "SEBASTIÁN": true, "SILVIO": true, "SIMON": true, "SIMÓN": true,
	"TEODORO": true, "TIMOTEO": true, "TOMAS": true, "TOMÁS": true, "VALENTIN": true,
	"VICENTE": true, "VLADIMIR": true, "WALTER": true, "WILLIAM": true, "WILLIAMS": true,
	"YAMIL": true, "YERKO": true, "YURI": true, "ZENON": true, "ALBERTO": true, "ALFREDO": true,
	"ALFONSO": true, "ARMANDO": true, "BENJAMIN": true, "BENJAMÍN": true, "CESAREO": true,
	"MARIA": true, "MARÍA": true, "ANA": true, "CARMEN": true, "ROSA": true, "JUANA": true,
	"ELENA": true, "PATRICIA": true, "CLAUDIA": true, "CARLA": true, "ANDREA": true,
	"DANIELA": true, "GABRIELA": true, "PAOLA": true, "PAMELA": true, "ALEJANDRA": true,
	"FABIOLA": true, "CECILIA": true, "VERONICA": true, "VERÓNICA": true, "SILVIA": true,
	"SANDRA": true, "MONICA": true, "MÓNICA": true, "LOURDES": true, "ELIZABETH": true,
	"BEATRIZ": true, "TERESA": true, "MARTHA": true, "MARTA": true, "SONIA": true,
	"MIRIAM": true, "NELLY": true, "GLADYS": true, "GLADIS": true, "VIRGINIA": true,
	"VICTORIA": true, "VALERIA": true, "VALENTINA": true, "CAMILA": true, "SOFIA": true,
	"SOFÍA": true, "LUCIA": true, "LUCÍA": true, "FERNANDA": true, "NATALIA": true,
	"NICOLE": true, "ADRIANA": true, "ALICIA": true, "AMALIA": true, "AMANDA": true,
	"ANGELA": true, "ÁNGELA": true, "ANGELICA": true, "ANGÉLICA": true, "ANTONIA": true,
	"ARACELI": true, "AURORA": true, "BELEN": true, "BELÉN": true, "BERTHA": true,
	"BLANCA": true, "BRENDA": true, "BRIGIDA": true, "CAROLINA": true, "CATALINA": true,
	"CELESTE": true, "CELIA": true, "CINTIA": true, "CINTHIA": true, "CYNTHIA": true,
	"CLARA": true, "CONCEPCION": true, "CONCEPCIÓN": true, "CONSUELO": true, "CRISTINA": true,
	"DALIA": true, "DAYANA": true, "DELIA": true, "DIANA": true, "DOLORES": true, "DORA": true,
	"EDITH": true, "ELIANA": true, "ELISA": true, "ELSA": true, "ELVIRA": true, "EMILIA": true,
	"ERIKA": true, "ESPERANZA": true, "ESTELA": true, "ESTHER": true, "EUGENIA": true,
	"EVA": true, "EVELYN": true, "FATIMA": true, "FÁTIMA": true, "FELIPA": true, "FLORA": true,
	"FRANCISCA": true, "GEOVANA": true, "GIOVANNA": true, "GLORIA": true, "GRACIELA": true,
	"GUADALUPE": true, "HORTENSIA": true, "INES": true, "INÉS": true, "IRENE": true, "IRIS": true,
	"ISABEL": true, "IVONNE": true, "JACQUELINE": true, "JANET": true, "JANETH": true,
	"JAQUELINE": true, "JESSICA": true, "JIMENA": true, "XIMENA": true, "JOSEFINA": true,
	"JUDITH": true, "JULIA": true, "JULIETA": true, "KAREN": true, "KARINA": true, "KARLA": true,
	"KATHERINE": true, "KATIA": true, "KELLY": true, "LETICIA": true, "LIDIA": true,
	"LILIAN": true, "LILIANA": true, "LINA": true, "LORENA": true, "LUISA": true, "LUZ": true,
	"MABEL": true, "MAGALI": true, "MAGALY": true, "MAGDALENA": true, "MARCELA": true,
	"MARGARITA": true, "MARIANA": true, "MARIBEL": true, "MARICELA": true, "MARINA": true,
	"MARISOL": true, "MARITZA": true, "MARLENE": true, "MARY": true, "MAYRA": true,
	"MELANIA": true, "MELINA": true, "MELISSA": true, "MERCEDES": true, "MICAELA": true,
	"MILENCA": true, "MILENKA": true, "MIREYA": true, "MIRNA": true, "NADIA": true,
	"NANCY": true, "NATIVIDAD": true, "NAYRA": true, "NEISA": true, "NILDA": true,
	"NOELIA": true, "NOEMI": true, "NOEMÍ": true, "NORA": true, "NORMA": true, "OLGA": true,
	"PAULINA": true, "PILAR": true, "PRISCILA": true, "RAQUEL": true, "REBECA": true,
	"REGINA": true, "REINA": true, "ROCIO": true, "ROCÍO": true, "ROMINA": true,
	"ROSARIO": true, "ROSMERY": true, "ROXANA": true, "RUTH": true, "SABINA": true,
	"SARA": true, "SHIRLEY": true, "SOLEDAD": true, "SUSANA": true, "TAMARA": true,
	"TANIA": true, "TATIANA": true, "URSULA": true, "VANESA": true, "VANESSA": true,
	"VIVIANA": true, "WANDA": true, "WENDY": true, "YANETH": true, "YANINA": true,
	"YESENIA": true, "YOLANDA": true, "YOVANA": true, "ZULEMA": true, "ZULMA": true,
}

// Palabras que aparecen en carnets bolivianos (ocupaciones, lugares administrativos, etiquetas)
// usadas en la ventana estructural y limpieza sin bloquear nombres reales como JULIO, MONTERO, MURILLO o TRINIDAD.
var boliviaNonNameWords = map[string]bool{
	"ESTUDIANTE": true, "COMERCIANTE": true, "INGENIERO": true, "INGENIERA": true,
	"ABOGADO": true, "ABOGADA": true, "MEDICO": true, "MÉDICO": true, "DOCENTE": true,
	"PROFESOR": true, "PROFESORA": true, "ARQUITECTO": true, "ARQUITECTA": true,
	"CONTADOR": true, "CONTADORA": true, "ADMINISTRADOR": true, "ADMINISTRADORA": true, "ECONOMISTA": true,
	"CHOFER": true, "CONDUCTOR": true, "MECANICO": true, "MECÁNICO": true, "ALBANIL": true,
	"ALBAÑIL": true, "CARPINTERO": true, "AGRICULTOR": true, "LABORES": true,
	"AMA": true, "EMPLEADO": true, "EMPLEADA": true, "INDEPENDIENTE": true, "JUBILADO": true,
	"JUBILADA": true, "RENTISTA": true, "ENFERMERA": true, "ENFERMERO": true, "ODONTOLOGO": true,
	"ODONTÓLOGO": true, "SISTEMAS": true, "CIVIL": true, "INDUSTRIAL": true, "COMERCIAL": true,
	"TECNICO": true, "TÉCNICO": true, "LICENCIADO": true, "LICENCIADA": true, "AUDITOR": true,
	"AUDITORA": true, "SECRETARIA": true, "SECRETARIO": true, "VENDEDOR": true, "VENDEDORA": true,
	"TRABAJADOR": true, "TRABAJADORA": true, "TRANSPORTISTA": true, "ARTESANO": true, "ARTESANA": true,
	"CONSULTOR": true, "CONSULTORA": true, "PROGRAMADOR": true, "DESARROLLADOR": true,
	"CERCADO": true, "QUILLACOLLO": true, "SACABA": true,
	"WARNES": true, "VIACHA": true, "YACUIBA": true, "RIBERALTA": true, "TUPIZA": true,
	"VILLAZON": true, "VILLAZÓN": true, "UYUNI": true, "ORURO": true, "POTOSI": true, "POTOSÍ": true,
	"TARIJA": true, "SUCRE": true, "CHUQUISACA": true, "COCHABAMBA": true,
	"BENI": true, "PANDO": true, "COBIJA": true, "PROVINCIA": true,
	"DEPARTAMENTO": true, "MUNICIPIO": true, "CANTON": true, "CANTÓN": true, "LOCALIDAD": true, "CIUDAD": true,
	"SECCION": true, "SECCIÓN": true, "SERIE": true,
	"REGISTRADO": true, "REGISTRADA": true, "NACIDO": true, "NACIDA": true, "NATURAL": true,
	"ENERO": true, "FEBRERO": true, "MARZO": true, "AGOSTO": true,
	"SEPTIEMBRE": true, "SETIEMBRE": true, "OCTUBRE": true, "NOVIEMBRE": true, "DICIEMBRE": true,
	"ZONA": true, "CALLE": true, "AVENIDA": true, "BARRIO": true, "URBANIZACION": true,
	"URBANIZACIÓN": true, "VILLA": true, "PASAJE": true, "MANZANO": true, "LOTE": true,
	"EDIFICIO": true, "PISO": true, "DEPTO": true, "URB": true, "AV": true, "NRO": true, "S/N": true,
}

var (
	// Etiquetas SEGIP ultra-tolerantes a errores OCR (0/O, 1/I/l, 8/B, 5/S, 3/E)
	reBolNombresLabel   = regexp.MustCompile(`(?i)\b[MN][O0]M[B8]R[E3][S5]?\b\s*[:.;-]?\s*(.*)$`)
	reBolApellidosLabel = regexp.MustCompile(`(?i)\bAP[E3][LI1]{1,2}[I1l][D0O][O0][S5]?\b\s*[:.;-]?\s*(.*)$`)
	reBolPrimerApLabel  = regexp.MustCompile(`(?i)\b(?:PR[I1l]M[E3]R\s+AP[E3][LI1]{1,2}[I1l][D0O][O0]|1\s*ER\.?\s+AP[E3][LI1]{1,2}[I1l][D0O][O0]|AP[E3][LI1]{1,2}[I1l][D0O][O0]\s+PAT[E3]RN[O0])\b\s*[:.;-]?\s*(.*)$`)
	reBolSegundoApLabel = regexp.MustCompile(`(?i)\b(?:S[E3]GUND[O0]\s+AP[E3][LI1]{1,2}[I1l][D0O][O0]|2\s*D[O0]\.?\s+AP[E3][LI1]{1,2}[I1l][D0O][O0]|AP[E3][LI1]{1,2}[I1l][D0O][O0]\s+MAT[E3]RN[O0])\b\s*[:.;-]?\s*(.*)$`)
	reBolClassicALabel  = regexp.MustCompile(`(?i)^\s*(?:A|4|Ä)\s*(?:[:.;-]\s*|\s+)(.*)$`)
	reBolPertenecenA    = regexp.MustCompile(`(?i)\bP[E3]RT[E3]N[E3]C[E3]N?\s+(?:A|4)\s*[:.;-]?\s*(.*)$`)
	reBolRegistrado     = regexp.MustCompile(`(?i)\bR[E3]G[I1l]STRAD[O0A@]\s*[:.;-]?\s*(.*)$`)
	reBolFirmaTitular   = regexp.MustCompile(`(?i)\b(F[I1l]RMA\s+D[E3]L\s+T[I1l]TULAR|F[I1l]RMA\s+T[I1l]TULAR|HUELLA\s+PULGAR|D[I1l]R[E3]CT[O0]R\s+G[E3]N[E3]RAL)\b`)
	reBolHeaderLine     = regexp.MustCompile(`(?i)\b(ESTAD[O0]\s+PLUR[I1l]NAC[I1l][O0]NAL|S[E3]RV[I1l]C[I1l][O0]\s+G[E3]N[E3]RAL|C[EÉ3]DULA\s+D[E3]\s+[I1l]D[E3]NT[I1l]DAD|S[E3]G[I1l]P|R[E3]PUBL[I1l]CA\s+D[E3]\s+B[O0]L[I1l]V[I1l]A|C[E3]RT[I1l]F[I1l]CA|B[I1l]C[E3]NT[E3]NAR[I1l][O0]|M[I1l]N[I1l]ST[E3]R[I1l][O0]\s+D[E3]\s+G[O0]B[I1l][E3]RN[O0]|P[O0]L[I1l]C[I1l]A\s+(?:NAC[I1l][O0]NAL|B[O0]L[I1l]V[I1l]ANA))\b`)
	reBolSkipMetaLine   = regexp.MustCompile(`(?i)\b(F[E3]CHA\s+D[E3]\s+NAC|NAC[I1l]D[O0]\s+EL|LUGAR\s+D[E3]\s+NAC|[O0]CUPAC[I1l][OÓ0]N|PR[O0]F[E3]S[I1l][OÓ0]N|ESTAD[O0]\s+C[I1l]V[I1l]L|D[O0]M[I1l]C[I1l]L[I1l][O0]|EM[I1l]S[I1l][OÓ0]N|EXP[I1l]RAC[I1l][OÓ0]N|V[E3]NC[I1l]M[I1l][E3]NT[O0]|VAL[I1l]D[E3]Z|VAL[I1l]D[AO]\s+HASTA|V[I1l]G[E3]NC[I1l]A|S[E3]R[I1l][E3]|S[E3]CC[I1l][OÓ0]N|NAC[I1l][O0]NAL[I1l]DAD|GRUP[O0]\s+SANGU[I1l]N[E3][O0]|F[O0]T[O0]GRAF[I1l]A|IMPR[E3]S[I1l][OÓ0]N)\b`)
	reBolStartsMetaLine = regexp.MustCompile(`(?i)^\s*(?:F[E3]CHA|NAC[I1l]D[O0]|LUGAR|[O0]CUPAC[I1l][OÓ0]N|PR[O0]F[E3]S[I1l][OÓ0]N|ESTAD[O0]\s+C[I1l]V[I1l]L|D[O0]M[I1l]C[I1l]L[I1l][O0]|EM[I1l]S[I1l][OÓ0]N|EXP[I1l]RAC[I1l][OÓ0]N|V[E3]NC[I1l]M[I1l][E3]NT[O0]|VAL[I1l]D[E3]Z|VAL[I1l]D[AO]|V[I1l]G[E3]NC[I1l]A|S[E3]R[I1l][E3]|S[E3]CC[I1l][OÓ0]N|NAC[I1l][O0]NAL[I1l]DAD|S[E3]X[O0]|G[EÉ3]N[E3]R[O0]|F[I1l]RMA|HUELLA|N[°ºo.*№]|C\.?\s*I\.?|NRO|GRUP[O0]|OBSERVAC[I1l][O0]NES|EN\s*[:.-])\b`)
	reBolAddressLine    = regexp.MustCompile(`(?i)^\s*(?:ZONA|CALLE|AV\.?|AVENIDA|BARRIO|URB\.?|URBANIZACI[OÓ]N|VILLA|PASAJE)\b`)
	reBolSerieLine      = regexp.MustCompile(`(?i)\b(?:S[E3]R[I1l][E3]|S[E3]CC[I1l][OÓ0]N|F[O0]L[I1l][O0]|PART[I1l]DA|L[I1l]BR[O0]|C[OÓ0]D[I1l]G[O0]|TEL[EÉ]F|CEL)\b`)
	reBolCINumber       = regexp.MustCompile(`(?i)\b(?:N[°º*№']|N[o0]\.(?:\s|$)|N[o0]\s*[:#-]|NRO\.?|C\.?\s*I\.?|C[EÉ]DULA(?:\s+DE\s+IDENTIDAD)?)\s*[:.#-]?\s*([0-9]{5,9}|[0-9]{1,3}(?:\.[0-9]{3}){1,2}|[0-9]{1,3}[ \t][0-9]{3}[ \t][0-9]{3})(?:[ \t]*-[ \t]*([0-9A-Z]{1,2})\b)?(?:[ \t]+(LP|LPZ|SC|SCZ|CB|CBA|OR|ORU|PT|PSI|TJ|TJA|CH|CHQ|BE|BN|PD|PND)\b)?`)
	reBolCIWithExt      = regexp.MustCompile(`(?i)\b([0-9]{5,9}(?:[ \t]*-[ \t]*[0-9A-Z]{1,2})?)[ \t]+(LP|LPZ|SC|SCZ|CB|CBA|OR|ORU|PT|PSI|TJ|TJA|CH|CHQ|BE|BN|PD|PND)\b`)
	reBolMRZLine1Fuzzy  = regexp.MustCompile(`(?:ID|I<|1D|I0|10)(?:BOL|B0L|8OL|80L)[<]?([0-9OQDILSZB]{5,9})`)
)

// ExtractBoliviaDocumentNumber extrae el número de C.I. boliviana evitando confundirlo con SERIE, SECCION o fechas.
func ExtractBoliviaDocumentNumber(lines []string, fullText string) string {
	// 1. Buscar etiqueta explícita N° / C.I. / CEDULA en líneas que NO sean de SERIE/SECCION/FECHA
	reExplicitCINumberLabel := regexp.MustCompile(`(?i)\b(?:N[°º*№']|N[o0]\.(?:\s|$)|N[o0]\s*[:#-]|NRO\.?|C\.?\s*I\.?\b)`)
	for i, line := range lines {
		if reBolSerieLine.MatchString(line) && !reExplicitCINumberLabel.MatchString(line) {
			continue
		}
		if m := reBolCINumber.FindStringSubmatch(line); m != nil {
			raw := m[1]
			if m[2] != "" {
				raw += "-" + m[2]
			}
			if m[3] != "" {
				raw += " " + m[3]
			}
			cand := normalizeDocNumberMatch(raw)
			if d := len(onlyDigits(cand)); d >= 5 && d <= 9 {
				return cand
			}
		}
		// Si la línea dice CEDULA DE IDENTIDAD / N° y el número está en la línea siguiente (sin SERIE/SECCION)
		if regexp.MustCompile(`(?i)\b(?:C[EÉ]DULA\s+DE\s+IDENTIDAD|N[°º*№']|N[o0]\.?|NRO\.?|C\.?\s*I\.?\b)\s*[:.#-]?\s*$`).MatchString(line) && i+1 < len(lines) {
			next := lines[i+1]
			if !reBolSerieLine.MatchString(next) && !reAnyDate.MatchString(next) {
				if m := reBolCIWithExt.FindStringSubmatch(next); m != nil {
					cand := normalizeDocNumberMatch(m[1] + " " + m[2])
					if d := len(onlyDigits(cand)); d >= 5 && d <= 9 {
						return cand
					}
				}
				if m := regexp.MustCompile(`\b([0-9]{5,9}(?:\s*-\s*[0-9A-Z]{1,2})?)\b`).FindStringSubmatch(next); m != nil {
					cand := normalizeDocNumberMatch(m[1])
					if d := len(onlyDigits(cand)); d >= 5 && d <= 9 {
						return cand
					}
				}
			}
		}
	}

	// 2. Número con sigla de departamento boliviano (ej. "8392011 LP" o "8392011-1A CB")
	for _, line := range lines {
		if reBolSerieLine.MatchString(line) {
			continue
		}
		if m := reBolCIWithExt.FindStringSubmatch(line); m != nil {
			cand := normalizeDocNumberMatch(m[1] + " " + m[2])
			if d := len(onlyDigits(cand)); d >= 5 && d <= 9 {
				return cand
			}
		}
	}

	// 3. Fallback: patrón en fullText completo sin SERIE
	if m := reBolCINumber.FindStringSubmatch(fullText); m != nil {
		raw := m[1]
		if m[2] != "" {
			raw += "-" + m[2]
		}
		if m[3] != "" {
			raw += " " + m[3]
		}
		cand := normalizeDocNumberMatch(raw)
		if d := len(onlyDigits(cand)); d >= 5 && d <= 9 {
			return cand
		}
	}

	// 4. Número suelto de 6-9 dígitos en una línea limpia (sin SERIE/SECCION/FECHA/barra)
	for _, line := range lines {
		if reBolSerieLine.MatchString(line) || reBolSkipMetaLine.MatchString(line) || reAnyDate.MatchString(line) || strings.Contains(line, "/") {
			continue
		}
		if m := reDoc3.FindStringSubmatch(line); m != nil {
			if d := len(onlyDigits(m[1])); d >= 6 && d <= 9 {
				return m[1]
			}
		}
	}
	return ""
}

func normalizeDocNumberMatch(raw string) string {
	s := strings.ToUpper(strings.TrimSpace(raw))
	s = regexp.MustCompile(`\s*-\s*`).ReplaceAllString(s, "-")
	// Quitar puntos/espacios internos dentro del bloque numérico principal
	parts := strings.Fields(s)
	if len(parts) == 0 {
		return ""
	}
	// Si los primeros tokens son solo dígitos agrupados (ej. "8" "392" "011" "LP" o "8.392.011" "LP")
	var numBuf strings.Builder
	idx := 0
	for idx < len(parts) {
		p := strings.ReplaceAll(parts[idx], ".", "")
		if regexp.MustCompile(`^[0-9]+(?:-[0-9A-Z]{1,2})?$`).MatchString(p) {
			if numBuf.Len() > 0 && (strings.Contains(numBuf.String(), "-") || len(onlyDigits(numBuf.String())) >= 5) {
				break
			}
			numBuf.WriteString(p)
			idx++
			if strings.Contains(p, "-") {
				break
			}
		} else {
			break
		}
	}
	if numBuf.Len() == 0 {
		clean := strings.ReplaceAll(parts[0], ".", "")
		return strings.Join(append([]string{clean}, parts[1:]...), " ")
	}
	rest := parts[idx:]
	if len(rest) > 0 {
		return strings.TrimSpace(numBuf.String() + " " + strings.Join(rest, " "))
	}
	return numBuf.String()
}

// IdentifyAndParseBoliviaDocument ejecuta el motor especializado para Cédula de Identidad de Bolivia (SEGIP).
func IdentifyAndParseBoliviaDocument(lines []string, fullText string) BoliviaDocumentAnalysis {
	analysis := BoliviaDocumentAnalysis{
		Fields: DocumentFields{
			DocumentType: "CI",
			Nationality:  "Boliviana",
		},
	}

	upFull := strings.ToUpper(fullText)
	isBol := reBolivia.MatchString(fullText) ||
		strings.Contains(upFull, "IDBOL") ||
		strings.Contains(upFull, "1DBOL") ||
		strings.Contains(upFull, "IDB0L") ||
		detectBoliviaCity(fullText) != "" ||
		reDoc2.MatchString(fullText)

	if !isBol && (rePeru.MatchString(fullText) || reArg.MatchString(fullText) || reChile.MatchString(fullText) || reBrasil.MatchString(fullText)) {
		return analysis
	}
	analysis.IsBolivia = true

	// Determinar sub-tipo de Cédula Boliviana
	if strings.Contains(upFull, "IDBOL") || strings.Contains(upFull, "IDB0L") || strings.Contains(upFull, "1DBOL") {
		analysis.Subtype = "SEGIP_MRZ_IDBOL"
	} else if reBolClassicALabel.MatchString(fullText) || reBolPertenecenA.MatchString(fullText) || reBolRegistrado.MatchString(fullText) || strings.Contains(upFull, "NACIDO EL") {
		analysis.Subtype = "SEGIP_CLASICO"
	} else {
		analysis.Subtype = "SEGIP_NUEVO"
	}

	// 1. Extraer Número de Cédula de Identidad Boliviana
	analysis.Fields.DocumentNumber = ExtractBoliviaDocumentNumber(lines, fullText)

	// 2. Extraer MRZ Boliviano (IDBOL) con decodificador tolerante a fallos de '<'
	mrzFields, hasBolMRZ := parseBoliviaIDBOLMRZ(lines)
	if hasBolMRZ {
		analysis.Subtype = "SEGIP_MRZ_IDBOL"
		if mrzFields.DocumentNumber != "" {
			analysis.Fields.DocumentNumber = mergeDocNumber(mrzFields.DocumentNumber, analysis.Fields.DocumentNumber)
		}
		if mrzFields.BirthDate != "" {
			analysis.Fields.BirthDate = mrzFields.BirthDate
		}
		if mrzFields.Gender != "" {
			analysis.Fields.Gender = mrzFields.Gender
		}
		if mrzFields.FullName != "" {
			analysis.Fields.FirstName = mrzFields.FirstName
			analysis.Fields.LastName = mrzFields.LastName
			analysis.Fields.FullName = mrzFields.FullName
			analysis.ExtractionMethod = "MRZ_IDBOL"
			analysis.ConfidenceScore = 98
		}
	}

	// 3. Extraer Nombres y Apellidos del Frente/Reverso con las estrategias SEGIP Bolivia
	if analysis.Fields.FullName == "" {
		first, last, full, method, score := extractBoliviaPersonName(lines)
		if full != "" {
			analysis.Fields.FirstName = first
			analysis.Fields.LastName = last
			analysis.Fields.FullName = full
			analysis.ExtractionMethod = method
			analysis.ConfidenceScore = score
		}
	}

	return analysis
}

// parseBoliviaIDBOLMRZ decodifica específicamente la zona MRZ de 3 líneas de los carnets de Bolivia (IDBOL),
// corrigiendo errores donde el OCR confunde '<' con 'K', 'C', 'L', o lee un solo '<' en vez de '<<'.
func parseBoliviaIDBOLMRZ(lines []string) (DocumentFields, bool) {
	var out DocumentFields
	detected := false

	replacer := strings.NewReplacer(" ", "", "\t", "", "«", "<", "»", "<", "‹", "<", "›", "<")
	for _, rawLine := range lines {
		norm := replacer.Replace(strings.ToUpper(rawLine))
		if len(norm) < 16 {
			continue
		}

		// Línea 1: IDBOL8392011<<4<<<<<<<<<<<<<<<
		if m := reBolMRZLine1Fuzzy.FindStringSubmatch(norm); m != nil {
			detected = true
			out.DocumentType = "CI"
			out.Nationality = "Boliviana"
			digits := onlyDigits(fixMRZDigits(m[1]))
			if len(digits) >= 5 && len(digits) <= 9 {
				out.DocumentNumber = digits
			}
			continue
		}

		// Línea 2: 9205148M3011205BOL<<<<<<<<<<<6
		if m := reTD1Line2.FindStringSubmatch(norm); m != nil {
			nat := fixMRZLetters(m[4])
			if nat == "BOL" || countryCodeMap[nat] != "" {
				detected = true
				if bDate := parseMRZDate(m[1]); bDate != "" {
					out.BirthDate = bDate
				}
				if m[2] == "M" || m[2] == "F" {
					out.Gender = m[2]
				}
				continue
			}
		}

		// Línea 3: APELLIDO1<APELLIDO2<<NOMBRE1<NOMBRE2<<<<<<
		// Nunca tratar Línea 1 ni Línea 2 (con >=4 dígitos reales o patrón BOL<<) como nombre.
		digitCount := len(onlyDigits(norm))
		if strings.Contains(norm, "<") &&
			!strings.Contains(norm, "IDBOL") &&
			!strings.Contains(norm, "1DBOL") &&
			!strings.Contains(norm, "IDB0L") &&
			!strings.Contains(norm, "BOL<<") &&
			!strings.HasPrefix(norm, "ID") &&
			!strings.HasPrefix(norm, "1D") &&
			!strings.HasPrefix(norm, "I<") &&
			!strings.HasPrefix(norm, "P<") &&
			digitCount <= 3 &&
			!reHas4Digit.MatchString(norm) {
			first, last, full := parseBoliviaMRZNameLine(norm)
			if len(full) >= 5 {
				detected = true
				out.FirstName = first
				out.LastName = last
				out.FullName = full
			}
		}
	}

	return out, detected
}

func safePrefix(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

// parseBoliviaMRZNameLine reconstruye Nombres y Apellidos de la 3ra línea MRZ de Bolivia:
// Soporta tanto "ROJAS<MAMANI<<MIJAEL<<<<<<" como lecturas degradadas "ROJAS<MAMANI<MIJAEL<<<<<<"
// y limpia sufijos 'KKK' o 'LLL' producidos cuando el OCR confunde '<<<' al final del nombre.
func parseBoliviaMRZNameLine(mrzLine string) (firstName, lastName, fullName string) {
	fixed := fixMRZLetters(mrzLine)
	var b strings.Builder
	for _, r := range fixed {
		if (r >= 'A' && r <= 'Z') || r == '<' {
			b.WriteRune(r)
		}
	}
	clean := strings.Trim(b.String(), "<")
	if clean == "" {
		return "", "", ""
	}

	// Caso A: Separador estándar '<<' entre bloque de apellidos y bloque de nombres
	if strings.Contains(clean, "<<") {
		parts := strings.Split(clean, "<<")
		rawSurnames := splitAndCleanMRZTokens(parts[0])
		var rawGiven []string
		for _, p := range parts[1:] {
			rawGiven = append(rawGiven, splitAndCleanMRZTokens(p)...)
		}
		if len(rawSurnames) > 0 && len(rawGiven) > 0 {
			last := toTitleCase(strings.Join(rawSurnames, " "))
			first := toTitleCase(strings.Join(rawGiven, " "))
			return first, last, strings.TrimSpace(first + " " + last)
		}
	}

	// Caso B: El OCR leyó solo un '<' entre apellidos y nombres (ej. "ROJAS<MAMANI<MIJAEL")
	tokens := splitAndCleanMRZTokens(clean)
	if len(tokens) < 2 {
		return "", "", ""
	}
	first, last := splitBoliviaTokensIntoFirstAndLast(tokens, true)
	if first == "" && last == "" {
		return "", "", ""
	}
	full := strings.TrimSpace(first + " " + last)
	return first, last, full
}

func splitAndCleanMRZTokens(segment string) []string {
	rawTokens := strings.Split(segment, "<")
	var out []string
	for _, tok := range rawTokens {
		t := strings.TrimSpace(tok)
		t = strings.TrimRight(t, "<")
		// 1. Si termina en "KK", "LL", "CC" (lectura errónea de "<<<"), recortar todas las letras repetidas del relleno MRZ
		if len(t) > 3 && (strings.HasSuffix(t, "KK") || strings.HasSuffix(t, "LL") || strings.HasSuffix(t, "CC")) {
			lastChar := t[len(t)-1]
			t = strings.TrimRight(t, string(lastChar))
		}
		// 2. Si aún termina en 'K', 'L' o 'C' pegada por un '<' final y sin esa letra coincide con un nombre o apellido boliviano real
		if len(t) > 4 && (strings.HasSuffix(t, "K") || strings.HasSuffix(t, "L") || strings.HasSuffix(t, "C")) {
			trimmed := t[:len(t)-1]
			if boliviaFirstNamesSet[trimmed] || boliviaSurnamesSet[trimmed] {
				t = trimmed
			}
		}
		if len(t) >= 2 && hasSpanishVowel(t) && !isForbiddenOrHeaderToken(t) {
			out = append(out, t)
		}
	}
	return out
}

// extractBoliviaPersonName aplica estrategias en cascada específicas para el Carnet de Bolivia (SEGIP).
func extractBoliviaPersonName(lines []string) (firstName, lastName, fullName, method string, score int) {
	var givenNames, surnames, firstSurname, secondSurname string
	givenIdx := -1
	surnameIdx := -1

	// helper que busca el valor en la misma línea o hasta 2 líneas hacia abajo.
	// NO aborta si la línea siguiente tiene columnas derechas (ej. "MIJAEL   14/05/1992" o "ROJAS MAMANI   EMISION: 10/11/2023");
	// solo corta si la línea COMIENZA con otra etiqueta/cabecera o no produce tokens limpios antes de la columna derecha.
	findValueAfterLabel := func(inline string, currentIdx int, maxWords int) (string, int) {
		val := sanitizeBoliviaNameTokens(inline, maxWords, false)
		if len(val) >= 2 {
			return val, currentIdx
		}
		for offset := 1; offset <= 2; offset++ {
			nextIdx := currentIdx + offset
			if nextIdx >= len(lines) {
				break
			}
			candLine := lines[nextIdx]
			if isPureBoliviaLabelLine(candLine) || reBolStartsMetaLine.MatchString(candLine) || reBolHeaderLine.MatchString(candLine) || reBolFirmaTitular.MatchString(candLine) {
				break
			}
			val = sanitizeBoliviaNameTokens(candLine, maxWords, false)
			if len(val) >= 2 {
				return val, nextIdx
			}
		}
		return "", -1
	}

	// ESTRATEGIA 1: Etiquetas SEGIP (NOMBRES, APELLIDOS, PRIMER APELLIDO, SEGUNDO APELLIDO)
	for i, line := range lines {
		// Caso especial SEGIP: "NOMBRES   APELLIDOS" en la misma fila de cabecera
		if reBolNombresLabel.MatchString(line) && reBolApellidosLabel.MatchString(line) && givenNames == "" && surnames == "" {
			if mN := reBolNombresLabel.FindStringSubmatch(line); mN != nil {
				// Si hay valores inline en la misma línea, los regex individuales los separarán abajo
				if sanitizeBoliviaNameTokens(mN[1], 3, false) == "" && i+1 < len(lines) {
					nextCand := sanitizeBoliviaNameTokens(lines[i+1], 4, false)
					if words := strings.Fields(nextCand); len(words) >= 2 {
						first, last := splitBoliviaTokensIntoFirstAndLast(words, false)
						if first != "" && last != "" {
							return first, last, strings.TrimSpace(first + " " + last), "ETIQUETAS_SEGIP", 94
						}
					}
				}
			}
		}

		if reBothApellidoHeaders.MatchString(line) && surnames == "" {
			if val, idx := findValueAfterLabel("", i, 3); val != "" {
				surnames = val
				surnameIdx = idx
			}
			continue
		}

		if m := reBolNombresLabel.FindStringSubmatch(line); m != nil && givenNames == "" {
			if val, idx := findValueAfterLabel(m[1], i, 3); val != "" {
				givenNames = val
				givenIdx = idx
			}
		}

		if m := reBolPrimerApLabel.FindStringSubmatch(line); m != nil && firstSurname == "" {
			if val, idx := findValueAfterLabel(m[1], i, 2); val != "" {
				firstSurname = val
				surnameIdx = idx
			}
		}

		if m := reBolSegundoApLabel.FindStringSubmatch(line); m != nil && secondSurname == "" {
			if val, _ := findValueAfterLabel(m[1], i, 2); val != "" {
				secondSurname = val
			}
		}

		if !reBolPrimerApLabel.MatchString(line) && !reBolSegundoApLabel.MatchString(line) {
			if m := reBolApellidosLabel.FindStringSubmatch(line); m != nil && surnames == "" {
				if val, idx := findValueAfterLabel(m[1], i, 3); val != "" {
					surnames = val
					surnameIdx = idx
				}
			}
		}
	}

	combinedSurnames := strings.TrimSpace(firstSurname + " " + secondSurname)
	if combinedSurnames == "" {
		combinedSurnames = surnames
	}
	if strings.EqualFold(givenNames, combinedSurnames) {
		combinedSurnames = ""
	}

	// Si encontró NOMBRES por etiqueta pero la etiqueta APELLIDOS se borró, buscar en las líneas vecinas
	if givenNames != "" && combinedSurnames == "" && givenIdx >= 0 {
		for _, candIdx := range []int{givenIdx + 1, givenIdx + 2, givenIdx - 1} {
			if candIdx >= 0 && candIdx < len(lines) && candIdx != givenIdx {
				raw := lines[candIdx]
				if !isPureBoliviaLabelLine(raw) && !reBolStartsMetaLine.MatchString(raw) && !reBolHeaderLine.MatchString(raw) && !reBolFirmaTitular.MatchString(raw) && !reDoc3.MatchString(raw) {
					cand := sanitizeBoliviaNameTokens(raw, 3, true)
					if cand != "" && !strings.EqualFold(cand, givenNames) {
						combinedSurnames = cand
						break
					}
				}
			}
		}
	}

	// Y si encontró APELLIDOS por etiqueta pero la etiqueta NOMBRES se borró, buscar en las líneas vecinas
	if combinedSurnames != "" && givenNames == "" && surnameIdx >= 0 {
		for _, candIdx := range []int{surnameIdx - 1, surnameIdx - 2, surnameIdx + 1} {
			if candIdx >= 0 && candIdx < len(lines) && candIdx != surnameIdx {
				raw := lines[candIdx]
				if !isPureBoliviaLabelLine(raw) && !reBolStartsMetaLine.MatchString(raw) && !reBolHeaderLine.MatchString(raw) && !reBolFirmaTitular.MatchString(raw) && !reDoc3.MatchString(raw) {
					cand := sanitizeBoliviaNameTokens(raw, 3, true)
					if cand != "" && !strings.EqualFold(cand, combinedSurnames) {
						givenNames = cand
						break
					}
				}
			}
		}
	}

	if givenNames != "" && combinedSurnames != "" {
		first := toTitleCase(givenNames)
		last := toTitleCase(combinedSurnames)
		return first, last, strings.TrimSpace(first + " " + last), "ETIQUETAS_SEGIP", 95
	}

	// ESTRATEGIA 2: Carnet Clásico de Bolivia ("A: ...", "PERTENECEN A: ...", "REGISTRADO: ...")
	for i, line := range lines {
		if m := reBolPertenecenA.FindStringSubmatch(line); m != nil {
			cand, _ := findValueAfterLabel(m[1], i, 4)
			if words := strings.Fields(cand); len(words) >= 2 {
				first, last := splitBoliviaTokensIntoFirstAndLast(words, false)
				return first, last, strings.TrimSpace(first + " " + last), "TITULAR_CLASICO_A", 93
			}
		}
		if m := reBolClassicALabel.FindStringSubmatch(line); m != nil {
			cand, _ := findValueAfterLabel(m[1], i, 4)
			if words := strings.Fields(cand); len(words) >= 2 {
				first, last := splitBoliviaTokensIntoFirstAndLast(words, false)
				return first, last, strings.TrimSpace(first + " " + last), "TITULAR_CLASICO_A", 92
			}
		}
		if m := reBolRegistrado.FindStringSubmatch(line); m != nil {
			cand, _ := findValueAfterLabel(m[1], i, 4)
			if words := strings.Fields(cand); len(words) >= 2 {
				first, last := splitBoliviaTokensIntoFirstAndLast(words, false)
				return first, last, strings.TrimSpace(first + " " + last), "TITULAR_REGISTRADO", 92
			}
		}
	}

	// ESTRATEGIA 3: Ventana Estructural y Clasificador Antroponímico de Bolivia (sin etiquetas)
	// Ubicar anclas de zona de nombre (después de CEDULA/N° y antes de FIRMA/NACIMIENTO/OCUPACION)
	docLineIdx := -1
	endZoneIdx := len(lines)
	for i, l := range lines {
		if docLineIdx == -1 && (reBolCINumber.MatchString(l) || reBolCIWithExt.MatchString(l) || strings.Contains(strings.ToUpper(l), "CEDULA") || strings.Contains(strings.ToUpper(l), "CÉDULA")) {
			docLineIdx = i
		}
		if reBolFirmaTitular.MatchString(l) || strings.Contains(strings.ToUpper(l), "NACIDO EL") {
			if i < endZoneIdx && i > 1 {
				endZoneIdx = i
			}
		}
	}

	type candidateLine struct {
		idx        int
		text       string
		words      []string
		firstScore int
		lastScore  int
		totalScore int
	}

	var candidates []candidateLine
	for i, line := range lines {
		if reBolHeaderLine.MatchString(line) || isPureBoliviaLabelLine(line) || reBolFirmaTitular.MatchString(line) {
			continue
		}
		if reBolStartsMetaLine.MatchString(line) || reBolAddressLine.MatchString(line) || reBolSerieLine.MatchString(line) {
			continue
		}
		if reTextDate.MatchString(line) || reDoc1.MatchString(line) || reDoc2.MatchString(line) {
			continue
		}
		if strings.Contains(line, "<<") || strings.Contains(strings.ToUpper(line), "IDBOL") || strings.Contains(strings.ToUpper(line), "1DBOL") {
			continue
		}
		// Si la línea es solo un número o fecha pura sin letras previas
		if reAnyDate.MatchString(strings.TrimSpace(line)) && sanitizeBoliviaNameTokens(line, 4, true) == "" {
			continue
		}

		clean := sanitizeBoliviaNameTokens(line, 4, true)
		words := strings.Fields(clean)
		if len(words) == 0 || len(words) > 4 {
			continue
		}

		fScore, lScore := scoreBoliviaPersonWords(words)
		if fScore > 0 || lScore > 0 || (len(words) >= 2 && len(words) <= 4 && allWordsLookLikeSpanishNames(words)) || (len(words) == 1 && len([]rune(words[0])) >= 4 && allWordsLookLikeSpanishNames(words)) {
			zoneBonus := 0
			if docLineIdx >= 0 && i > docLineIdx && i < endZoneIdx {
				zoneBonus = 8
			}
			candidates = append(candidates, candidateLine{
				idx:        i,
				text:       clean,
				words:      words,
				firstScore: fScore,
				lastScore:  lScore,
				totalScore: fScore + lScore + len(words)*3 + zoneBonus,
			})
		}
	}

	// Si hay una sola línea candidata con 2-4 palabras
	if len(candidates) == 1 && len(candidates[0].words) >= 2 {
		first, last := splitBoliviaTokensIntoFirstAndLast(candidates[0].words, false)
		return first, last, strings.TrimSpace(first + " " + last), "VENTANA_ESTRUCTURAL_BOLIVIA", 88
	}

	// Elegir primero si una sola línea tiene >= 3 palabras con nombre Y apellido conocidos y el mayor puntaje
	bestSingleIdx := -1
	bestSingleScore := 0
	for idx, c := range candidates {
		if len(c.words) >= 3 && c.firstScore > 0 && c.lastScore > 0 && c.totalScore > bestSingleScore {
			bestSingleScore = c.totalScore
			bestSingleIdx = idx
		}
	}

	// Buscar el mejor par adyacente (gap de líneas <= 3) y mejor triplete adyacente
	bestPairScore := -1
	bestPairA, bestPairB := -1, -1
	for i := 0; i < len(candidates); i++ {
		for j := i + 1; j < len(candidates); j++ {
			gap := candidates[j].idx - candidates[i].idx
			if gap > 3 {
				continue
			}
			pairScore := candidates[i].totalScore + candidates[j].totalScore
			if gap == 1 {
				pairScore += 10
			} else if gap == 2 {
				pairScore += 5
			}
			if (candidates[i].firstScore > 0 && candidates[j].lastScore > 0) || (candidates[i].lastScore > 0 && candidates[j].firstScore > 0) {
				pairScore += 12
			}
			if pairScore > bestPairScore {
				bestPairScore = pairScore
				bestPairA, bestPairB = i, j
			}
		}
	}

	// Verificar triplete de 1-1-1/2 palabras consecutivas (PRIMER APELLIDO / SEGUNDO APELLIDO / NOMBRES o viceversa)
	if len(candidates) >= 3 {
		for i := 0; i+2 < len(candidates); i++ {
			c0, c1, c2 := candidates[i], candidates[i+1], candidates[i+2]
			if c2.idx-c0.idx <= 4 && len(c0.words) == 1 && len(c1.words) == 1 && len(c2.words) >= 1 {
				tripleScore := c0.totalScore + c1.totalScore + c2.totalScore + 10
				if tripleScore >= bestPairScore && tripleScore >= bestSingleScore {
					if c0.firstScore > c0.lastScore && c2.lastScore >= c2.firstScore {
						first := toTitleCase(c0.text)
						last := toTitleCase(c1.text + " " + c2.text)
						return first, last, strings.TrimSpace(first + " " + last), "VENTANA_ESTRUCTURAL_BOLIVIA", 90
					}
					if c2.firstScore >= c2.lastScore && (c0.lastScore > 0 || c1.lastScore > 0) {
						first := toTitleCase(c2.text)
						last := toTitleCase(c0.text + " " + c1.text)
						return first, last, strings.TrimSpace(first + " " + last), "VENTANA_ESTRUCTURAL_BOLIVIA", 90
					}
				}
			}
		}
	}

	if bestSingleIdx >= 0 && (bestPairScore < 0 || bestSingleScore >= bestPairScore) {
		c := candidates[bestSingleIdx]
		first, last := splitBoliviaTokensIntoFirstAndLast(c.words, false)
		return first, last, strings.TrimSpace(first + " " + last), "VENTANA_ESTRUCTURAL_BOLIVIA", 91
	}

	if bestPairA >= 0 && bestPairB >= 0 {
		c0, c1 := candidates[bestPairA], candidates[bestPairB]
		// Asegurar que al menos uno tenga puntaje antroponímico si ambos son de 1 sola palabra sin puntaje
		if c0.firstScore+c0.lastScore+c1.firstScore+c1.lastScore > 0 || len(c0.words)+len(c1.words) >= 3 {
			c0IsSurname := c0.lastScore > c0.firstScore && c1.firstScore >= c1.lastScore
			if c0IsSurname {
				first := toTitleCase(c1.text)
				last := toTitleCase(c0.text)
				return first, last, strings.TrimSpace(first + " " + last), "VENTANA_ESTRUCTURAL_BOLIVIA", 90
			}
			first := toTitleCase(c0.text)
			last := toTitleCase(c1.text)
			return first, last, strings.TrimSpace(first + " " + last), "VENTANA_ESTRUCTURAL_BOLIVIA", 90
		}
	}

	if len(candidates) >= 1 && len(candidates[0].words) >= 2 && (candidates[0].firstScore > 0 || candidates[0].lastScore > 0) {
		first, last := splitBoliviaTokensIntoFirstAndLast(candidates[0].words, false)
		return first, last, strings.TrimSpace(first + " " + last), "VENTANA_ESTRUCTURAL_BOLIVIA", 86
	}

	// Si solo habíamos encontrado givenNames o combinedSurnames parcialmente en Estrategia 1
	if givenNames != "" || combinedSurnames != "" {
		first := toTitleCase(givenNames)
		last := toTitleCase(combinedSurnames)
		return first, last, strings.TrimSpace(first + " " + last), "ETIQUETAS_SEGIP_PARCIAL", 75
	}

	return "", "", "", "", 0
}

func isAnyBoliviaLabelLine(line string) bool {
	return reBolNombresLabel.MatchString(line) ||
		reBolApellidosLabel.MatchString(line) ||
		reBolPrimerApLabel.MatchString(line) ||
		reBolSegundoApLabel.MatchString(line) ||
		reBothApellidoHeaders.MatchString(line)
}

func isPureBoliviaLabelLine(line string) bool {
	if !isAnyBoliviaLabelLine(line) {
		return false
	}
	// Si es etiqueta pero además contiene valor de nombre en la misma línea, no es etiqueta pura
	if m := reBolNombresLabel.FindStringSubmatch(line); m != nil && sanitizeBoliviaNameTokens(m[1], 3, false) != "" {
		return false
	}
	if m := reBolApellidosLabel.FindStringSubmatch(line); m != nil && sanitizeBoliviaNameTokens(m[1], 3, false) != "" {
		return false
	}
	if m := reBolPrimerApLabel.FindStringSubmatch(line); m != nil && sanitizeBoliviaNameTokens(m[1], 2, false) != "" {
		return false
	}
	if m := reBolSegundoApLabel.FindStringSubmatch(line); m != nil && sanitizeBoliviaNameTokens(m[1], 2, false) != "" {
		return false
	}
	return true
}

func sanitizeBoliviaNameTokens(raw string, maxWords int, filterNonNames bool) string {
	base := sanitizePersonTokens(raw, maxWords)
	if base == "" {
		return ""
	}
	if !filterNonNames {
		return base
	}
	words := strings.Fields(base)
	var filtered []string
	for _, w := range words {
		up := strings.ToUpper(w)
		if boliviaNonNameWords[up] || boliviaNonNameWords[foldAccents(up)] {
			continue
		}
		filtered = append(filtered, up)
	}
	return strings.Join(filtered, " ")
}

func scoreBoliviaPersonWords(words []string) (firstScore, lastScore int) {
	for _, w := range words {
		up := strings.ToUpper(w)
		folded := foldAccents(up)
		if boliviaFirstNamesSet[up] || boliviaFirstNamesSet[folded] {
			firstScore += 15
		}
		if boliviaSurnamesSet[up] || boliviaSurnamesSet[folded] {
			lastScore += 15
		}
		// Sufijos patronímicos hispanos/andinos típicos de apellidos en Bolivia
		if strings.HasSuffix(folded, "EZ") || strings.HasSuffix(folded, "ANI") || strings.HasSuffix(folded, "ANA") && len(folded) >= 6 || strings.HasSuffix(up, "AÑA") || strings.HasSuffix(folded, "UIA") {
			lastScore += 4
		}
	}
	return firstScore, lastScore
}

func allWordsLookLikeSpanishNames(words []string) bool {
	for _, w := range words {
		up := strings.ToUpper(w)
		folded := foldAccents(up)
		if len([]rune(up)) < 3 || boliviaNonNameWords[up] || boliviaNonNameWords[folded] || isForbiddenOrHeaderToken(up) {
			return false
		}
		// Evitar siglas o fragmentos con muchas consonantes seguidas
		consonantRun := 0
		for _, r := range up {
			if !strings.ContainsRune("AEIOUÁÉÍÓÚÜY", r) {
				consonantRun++
				if consonantRun >= 4 {
					return false
				}
			} else {
				consonantRun = 0
			}
		}
	}
	return true
}

// splitBoliviaTokensIntoFirstAndLast separa una lista de palabras (ej. ["MIJAEL", "ROJAS", "MAMANI"] o ["ROJAS", "MAMANI", "MIJAEL"])
// en (FirstName, LastName) usando la base léxica de Bolivia y el orden del documento (preferSurnamesFirst=true en MRZ).
func splitBoliviaTokensIntoFirstAndLast(tokens []string, preferSurnamesFirst bool) (firstName, lastName string) {
	var clean []string
	for _, t := range tokens {
		up := strings.ToUpper(strings.TrimSpace(t))
		if up != "" && !isForbiddenOrHeaderToken(up) && !boliviaNonNameWords[up] {
			clean = append(clean, up)
		}
	}
	n := len(clean)
	if n == 0 {
		return "", ""
	}
	if n == 1 {
		w := toTitleCase(clean[0])
		if boliviaSurnamesSet[clean[0]] && !boliviaFirstNamesSet[clean[0]] {
			return "", w
		}
		return w, ""
	}
	if n == 2 {
		w0, w1 := clean[0], clean[1]
		if (boliviaSurnamesSet[w0] && boliviaFirstNamesSet[w1]) || (preferSurnamesFirst && !boliviaFirstNamesSet[w0]) {
			return toTitleCase(w1), toTitleCase(w0)
		}
		return toTitleCase(w0), toTitleCase(w1)
	}
	if n == 3 {
		w0, w1, w2 := clean[0], clean[1], clean[2]
		if preferSurnamesFirst {
			if boliviaFirstNamesSet[w0] && boliviaSurnamesSet[w2] && !boliviaSurnamesSet[w0] {
				return toTitleCase(w0), toTitleCase(w1 + " " + w2)
			}
			return toTitleCase(w2), toTitleCase(w0 + " " + w1)
		}
		if boliviaSurnamesSet[w1] && !boliviaFirstNamesSet[w1] {
			return toTitleCase(w0), toTitleCase(w1 + " " + w2)
		}
		if boliviaSurnamesSet[w0] && boliviaSurnamesSet[w1] && boliviaFirstNamesSet[w2] {
			return toTitleCase(w2), toTitleCase(w0 + " " + w1)
		}
		if boliviaFirstNamesSet[w1] && !boliviaSurnamesSet[w1] {
			return toTitleCase(w0 + " " + w1), toTitleCase(w2)
		}
		return toTitleCase(w0), toTitleCase(w1 + " " + w2)
	}

	if preferSurnamesFirst {
		return toTitleCase(strings.Join(clean[2:], " ")), toTitleCase(strings.Join(clean[:2], " "))
	}
	if boliviaSurnamesSet[clean[0]] && boliviaSurnamesSet[clean[1]] && boliviaFirstNamesSet[clean[2]] {
		return toTitleCase(strings.Join(clean[2:], " ")), toTitleCase(strings.Join(clean[:2], " "))
	}
	return toTitleCase(strings.Join(clean[:2], " ")), toTitleCase(strings.Join(clean[2:], " "))
}

// InferGenderFromBoliviaFirstName deduce el sexo ("M" o "F") a partir del nombre de pila si el carnet no lo trae explícito en el frente.
func InferGenderFromBoliviaFirstName(firstName string) string {
	words := strings.Fields(strings.ToUpper(strings.TrimSpace(firstName)))
	if len(words) == 0 {
		return ""
	}
	first := words[0]
	femaleExceptions := map[string]bool{
		"CARMEN": true, "LOURDES": true, "BELEN": true, "BELÉN": true, "CONCEPCION": true,
		"CONCEPCIÓN": true, "CONSUELO": true, "DOLORES": true, "EDITH": true, "ESTHER": true,
		"INES": true, "INÉS": true, "IRIS": true, "ISABEL": true, "JANET": true, "JANETH": true,
		"JUDITH": true, "YUDITH": true, "KAREN": true, "LILIAN": true, "LUZ": true, "MABEL": true,
		"MAGALI": true, "MAGALY": true, "MARIBEL": true, "MARISEL": true, "MARISOL": true, "MARY": true,
		"MERCEDES": true, "MIRIAM": true, "NATIVIDAD": true, "NELLY": true, "NOEMI": true, "NOEMÍ": true,
		"NOHEMI": true, "PILAR": true, "RAQUEL": true, "ROCIO": true, "ROCÍO": true, "ROSARIO": true,
		"ROSMERY": true, "RUTH": true, "SHIRLEY": true, "SOLEDAD": true, "YANETH": true,
		"GLADYS": true, "GLADIS": true, "EVELYN": true, "NICOLE": true, "WENDY": true,
		"BEATRIZ": true, "ELIZABETH": true, "IRENE": true, "CELESTE": true, "GUADALUPE": true,
		"IVONNE": true, "JACQUELINE": true, "JAQUELINE": true, "KATHERINE": true, "MARLENE": true,
		"ARACELI": true, "ANAHI": true, "ANAHÍ": true, "LIZETH": true, "INGRID": true,
		"ASTRID": true, "FLOR": true, "KELLY": true, "DULCE": true, "NAYELI": true, "NAYELY": true,
		"RUBI": true, "RUBÍ": true, "DAISY": true, "MELANIE": true, "STEPHANIE": true,
	}
	if femaleExceptions[first] {
		return "F"
	}
	if strings.HasSuffix(first, "A") && first != "LUCA" && first != "JOSHUA" && first != "BORJA" {
		return "F"
	}
	if boliviaFirstNamesSet[first] {
		return "M"
	}
	var lastRune rune
	for _, r := range first {
		lastRune = unicode.ToUpper(r)
	}
	if lastRune == 'O' || lastRune == 'L' || lastRune == 'N' || lastRune == 'R' || lastRune == 'S' {
		return "M"
	}
	return ""
}
