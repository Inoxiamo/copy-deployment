#!/bin/bash

# Script per duplicare un deployment Kubernetes con suffisso "-test"

# Valori di default
DEFAULT_NAMESPACE="namespace-test"
DEFAULT_DEPLOYMENT_NAME="deployment-test"

# Funzione per mostrare l'uso dello script
usage() {
    echo "Uso: $0 [-n namespace] [-d deployment_name] [-l label_key]"
    echo "  -n namespace         Namespace del deployment (default: $DEFAULT_NAMESPACE)"
    echo "  -d deployment_name   Nome del deployment da duplicare (default: $DEFAULT_DEPLOYMENT_NAME)"
    echo "  -l label_key         Chiave dell'etichetta da utilizzare (default: app)"
    exit 1
}

# Valore di default per la chiave dell'etichetta
LABEL_KEY="app"

# Parsing degli argomenti da linea di comando
while getopts ":n:d:l:h" opt; do
  case $opt in
    n)
      NAMESPACE="$OPTARG"
      ;;
    d)
      ORIGINAL_DEPLOYMENT_NAME="$OPTARG"
      ;;
    l)
      LABEL_KEY="$OPTARG"
      ;;
    h)
      usage
      ;;
    \?)
      echo "Opzione non valida: -$OPTARG" >&2
      usage
      ;;
    :)
      echo "L'opzione -$OPTARG richiede un argomento." >&2
      usage
      ;;
  esac
done

# Imposta i valori di namespace e deployment se non specificati
NAMESPACE="${NAMESPACE:-$DEFAULT_NAMESPACE}"
ORIGINAL_DEPLOYMENT_NAME="${ORIGINAL_DEPLOYMENT_NAME:-$DEFAULT_DEPLOYMENT_NAME}"

NEW_DEPLOYMENT_NAME="${ORIGINAL_DEPLOYMENT_NAME}-TEST"

# Verifica la connessione al cluster
if ! kubectl cluster-info > /dev/null 2>&1; then
    echo "Errore: impossibile connettersi al cluster Kubernetes." >&2
    exit 1
fi

# Verifica se il namespace esiste
if ! kubectl get namespace "$NAMESPACE" > /dev/null 2>&1; then
    echo "Errore: il namespace '$NAMESPACE' non esiste." >&2
    exit 1
fi

# Verifica se il deployment originale esiste
if ! kubectl get deployment "$ORIGINAL_DEPLOYMENT_NAME" -n "$NAMESPACE" > /dev/null 2>&1; then
    echo "Errore: il deployment '$ORIGINAL_DEPLOYMENT_NAME' non esiste nel namespace '$NAMESPACE'." >&2
    exit 1
fi

# Estrai il deployment originale in formato YAML
kubectl get deployment "$ORIGINAL_DEPLOYMENT_NAME" -n "$NAMESPACE" -o yaml | sed '/^status:$/,/^[^ ]/d' > ./original-deployment.yaml

# Rimuovi campi gestiti dal sistema utilizzando yq
yq e 'del(.metadata.uid, .metadata.resourceVersion, .metadata.creationTimestamp, .metadata.generation, .metadata.managedFields)' -i ./original-deployment.yaml

# Modifica il nome del deployment
yq e ".spec.replicas = 1" -i ./original-deployment.yaml
yq e ".metadata.name = \"$NEW_DEPLOYMENT_NAME\"" -i ./original-deployment.yaml

# Rimuovi il servizio se presente
# Mantieni le porte esposte
# Mantieni le porte esposte (rimuovi l'eliminazione delle porte)

# Mantieni le configurazioni necessarie come serviceAccount, configMap, secrets, immagini, ecc.
# Le seguenti righe preservano alcune parti importanti del deployment come il serviceAccount, configMap, secret, e image, mentre sovrascrivono solo le sezioni necessarie.

# Estrai il valore dell'etichetta originale
ORIGINAL_APP_LABEL=$(yq e ".spec.template.metadata.labels.$LABEL_KEY" ./original-deployment.yaml)

# Verifica se l'etichetta esiste
if [ "$ORIGINAL_APP_LABEL" = "null" ] || [ -z "$ORIGINAL_APP_LABEL" ]; then
    echo "Errore: l'etichetta '$LABEL_KEY' non esiste nel deployment originale." >&2
    rm ./original-deployment.yaml
    exit 1
fi

# Crea il nuovo valore dell'etichetta
NEW_APP_LABEL="${ORIGINAL_APP_LABEL}-test"

# Modifica le etichette dei pod e il selector del deployment
yq e ".spec.template.metadata.labels.$LABEL_KEY = \"$NEW_APP_LABEL\"" -i ./original-deployment.yaml
yq e ".spec.selector.matchLabels.$LABEL_KEY = \"$NEW_APP_LABEL\"" -i ./original-deployment.yaml

# Applica il nuovo deployment
if ! kubectl apply -f ./original-deployment.yaml -n "$NAMESPACE"; then
    echo "Errore: si è verificato un problema durante l'applicazione del nuovo deployment." >&2
    rm ./original-deployment.yaml
    exit 1
fi

# Pulizia del file temporaneo
rm ./original-deployment.yaml

echo "Deployment duplicato con successo come '$NEW_DEPLOYMENT_NAME' nel namespace '$NAMESPACE'."