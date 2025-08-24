#!/bin/bash

# Script para compilar LocIp para múltiples plataformas
# Uso: ./build.sh

set -e

echo "🔨 Compilando LocIp para múltiples plataformas..."

# Crear directorio de builds si no existe
mkdir -p builds

# Limpiar builds anteriores
rm -rf builds/*

# Función para compilar
build_for_platform() {
    local GOOS=$1
    local GOARCH=$2
    local EXTENSION=$3
    local OUTPUT="builds/locip_${GOOS}_${GOARCH}${EXTENSION}"
    
    echo "📦 Compilando para ${GOOS}/${GOARCH}..."
    
    GOOS=$GOOS GOARCH=$GOARCH go build \
        -ldflags="-s -w" \
        -o "$OUTPUT" .
    
    echo "✅ Creado: $OUTPUT"
}

# Compilar para diferentes plataformas
build_for_platform "darwin" "amd64" ""      # macOS Intel
build_for_platform "darwin" "arm64" ""      # macOS Apple Silicon
build_for_platform "linux" "amd64" ""       # Linux x64
build_for_platform "linux" "arm64" ""       # Linux ARM64
build_for_platform "windows" "amd64" ".exe" # Windows x64
build_for_platform "windows" "arm64" ".exe" # Windows ARM64

# Crear checksums
echo "🔍 Generando checksums..."
cd builds
for file in *; do
    if [[ -f "$file" ]]; then
        shasum -a 256 "$file" > "${file}.sha256"
        echo "📝 Checksum creado para: $file"
    fi
done

# Mostrar resumen
echo ""
echo "🎉 ¡Compilación completada!"
echo "📁 Archivos creados en el directorio 'builds/':"
ls -la builds/
echo ""
echo "💡 Para instalar en tu sistema:"
echo "   cp builds/locip_darwin_$(uname -m) /usr/local/bin/locip"
echo "   chmod +x /usr/local/bin/locip"
