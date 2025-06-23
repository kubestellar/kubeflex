package util

import (
	"context"
	"fmt"
	"strings"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/jackc/pgx/v5"
)

const (
	pgAdminDB        = "postgres"
	pgCredsSecretKey = "postgres-password"
)

func DropDatabase(ctx context.Context, cpName string, crClient client.Client) error {
	dbPass, err := GetPGDBPassword(crClient)
	if err != nil {
		return err
	}
	connStr := GeneratePGConnectionString(dbPass, pgAdminDB)

	conn, err := pgx.Connect(ctx, connStr)
	if err != nil {
		return fmt.Errorf("unable to connect to database: %s", err)
	}
	defer conn.Close(ctx)

	dbName := ReplaceNotAllowedCharsInDBName(cpName)
	sqlSTM := fmt.Sprintf("DROP DATABASE IF EXISTS %s WITH (FORCE)", dbName)
	_, err = conn.Exec(ctx, sqlSTM)
	if err != nil {
		return fmt.Errorf("error deleting DB: %s", err)
	}

	return nil
}

// TODO: change func signature to be more explicit
func ReplaceNotAllowedCharsInDBName(name string) string {
	return strings.ReplaceAll(name, "-", "_")
}

func GeneratePSecretName(releaseName string) string {
	return fmt.Sprintf("%s-postgresql", releaseName)
}

func GeneratePSReplicaSetName(releaseName string) string {
	return fmt.Sprintf("%s-postgresql", releaseName)
}

func GeneratePGConnectionString(dbPassword, dbName string) string {
	return fmt.Sprintf("postgres://postgres:%s@%s-postgresql.%s.svc/%s?sslmode=disable", dbPassword, DBReleaseName, SystemNamespace, dbName)
}

func GetPGDBPassword(crClient client.Client) (string, error) {
	pSecret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      GeneratePSecretName(DBReleaseName),
			Namespace: SystemNamespace,
		},
	}

	err := crClient.Get(context.TODO(), client.ObjectKeyFromObject(pSecret), pSecret, &client.GetOptions{})
	if err != nil {
		return "", err
	}

	return string(pSecret.Data[pgCredsSecretKey]), nil
}
