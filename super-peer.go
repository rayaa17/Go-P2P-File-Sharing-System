package main

import (
	"database/sql"
	"fmt"
	"log"
	"net"
	"strings"

	_ "github.com/go-sql-driver/mysql"
)

var db *sql.DB

func main() {
	// Initialise la base de données MySQL
	var err error
	db, err = sql.Open("mysql", "root:rayane@tcp(localhost:3306)/srtp")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Crée la table PAIRES si elle n'existe pas déjà
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS PAIRES (
                        id INT AUTO_INCREMENT PRIMARY KEY,
                        ip VARCHAR(255),
                        fichier VARCHAR(255) 
                        )`)
	if err != nil {
		log.Fatal(err)
	}

	// Démarre le serveur TCP sur le port 8077
	listener, err := net.Listen("tcp", ":8077")
	if err != nil {
		fmt.Println("Erreur lors de l'écoute:", err)
		return
	}
	defer listener.Close()

	fmt.Println("Super pair en attente de connexion sur le port 8077...")

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Erreur lors de l'acceptation de la connexion:", err)
			continue
		}

		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()

	var buffer [1024]byte
	n, err := conn.Read(buffer[:])
	if err != nil {
		fmt.Println("Erreur lors de la lecture:", err)
		return
	}

	request := strings.TrimSpace(string(buffer[:n]))

	switch request {
	case "publish":
		handlePublish(conn)
	case "search":
		handleSearch(conn)
	case "request":
		handleRequest(conn)
	default:
		fmt.Println("Requête non reconnue:", request)
	}
}

func handlePublish(conn net.Conn) {
	var buffer [1024]byte
	n, err := conn.Read(buffer[:])
	if err != nil {
		fmt.Println("Erreur lors de la lecture des informations de publication:", err)
		return
	}

	data := strings.Split(strings.TrimSpace(string(buffer[:n])), ";")
	if len(data) != 2 {
		fmt.Println("Données de publication incorrectes:", data)
		return
	}

	ip, fileName := data[0], data[1]
	_, err = db.Exec("INSERT INTO PAIRES (ip, fichier) VALUES (?, ?)", ip, fileName)
	if err != nil {
		fmt.Println("Erreur lors de l'insertion dans la base de données:", err)
		return
	}
	fmt.Println("Client enregistré :", ip)

	conn.Write([]byte("Fichier publié avec succès"))
}

func handleSearch(conn net.Conn) {
	var buffer [1024]byte
	n, err := conn.Read(buffer[:])
	if err != nil {
		fmt.Println("Erreur lors de la lecture des informations de recherche:", err)
		return
	}

	fileName := strings.TrimSpace(string(buffer[:n]))
	ip, err := searchFile(fileName)
	if err != nil {
		if err == sql.ErrNoRows {
			conn.Write([]byte("not_found"))
		} else {
			fmt.Println("Erreur lors de la recherche du fichier dans la base de données:", err)
		}
		return
	}

	conn.Write([]byte(ip))
}

func handleRequest(conn net.Conn) {
	var buffer [1024]byte
	n, err := conn.Read(buffer[:])
	if err != nil {
		fmt.Println("Erreur lors de la lecture des informations de demande:", err)
		return
	}

	fileName := strings.TrimSpace(string(buffer[:n]))
	ip, err := searchFile(fileName)
	if err != nil {
		if err == sql.ErrNoRows {
			conn.Write([]byte("not_found"))
		} else {
			fmt.Println("Erreur lors de la recherche du fichier dans la base de données:", err)
		}
		return
	}

	conn.Write([]byte(ip))
}

func searchFile(fileName string) (string, error) {
	var ip string
	err := db.QueryRow("SELECT ip FROM PAIRES WHERE fichier = ?", fileName).Scan(&ip)
	if err != nil {
		return "", err
	}
	return ip, nil
}
