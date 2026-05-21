package gitlab

import (
    "crypto/x509"
    "fmt"
    "os"
)

func loadCACertPool(caFile string) (*x509.CertPool, error) {
    data, err := os.ReadFile(caFile)
    if err != nil {
        return nil, fmt.Errorf("unable to read CA file: %w", err)
    }
    pool := x509.NewCertPool()
    if ok := pool.AppendCertsFromPEM(data); !ok {
        return nil, fmt.Errorf("failed to parse CA bundle from %s", caFile)
    }
    return pool, nil
}
