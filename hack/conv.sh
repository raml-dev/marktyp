#!/bin/bash

# --- CONFIGURAZIONE ---
# Assicurati che il file sia in questa cartella
INPUT="icon/icon.png" 
# Nome base senza lo slash iniziale
OUTPUT_NAME="marktyp_icon" 

# Verifica esistenza file
if [ ! -f "$INPUT" ]; then
    echo "Errore: Non trovo $INPUT"
    exit 1
fi

echo "Elaborazione in corso..."

# 1. Creazione SVG (Sintassi semplificata per evitare crash del parser)
vtracer --input "$INPUT" --output "${OUTPUT_NAME}.svg"

# 2. Generazione formati PNG (Usiamo ./ per forzare la cartella corrente)
SIZES=(16 32 48 64 128 256 512 1024)

for SIZE in "${SIZES[@]}"; do
    magick "$INPUT" -resize "${SIZE}x${SIZE}" "./${OUTPUT_NAME}_${SIZE}.png"
done

# 3. Creazione file .ico (Favicon)
magick "$INPUT" -define icon:auto-resize=16,32,48,64,128,256 "./${OUTPUT_NAME}.ico"

echo "Completato! I file sono stati creati nella cartella corrente."
ls -1 ${OUTPUT_NAME}*