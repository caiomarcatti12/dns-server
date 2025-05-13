#!/bin/bash

# Script para compilar servidor DNS em Golang para Linux

# Nome do binário final
OUTPUT="dns_server"

# Verifica se Go está instalado
if ! command -v go &> /dev/null
then
    echo "Erro: Golang não está instalado ou não está no PATH."
    exit 1
fi

# Baixa dependências
echo "Baixando dependências..."
go mod init dns_server &> /dev/null
go mod tidy

# Compilação para Linux
GOOS=linux GOARCH=amd64 go build -o "$OUTPUT" main.go

if [ $? -eq 0 ]; then
    echo "Compilação concluída com sucesso! Binário gerado: ./$OUTPUT"
else
    echo "Erro durante a compilação."
fi
