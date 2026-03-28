package main

import (
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
)

func main() {
	var port string
	var fileName string

	// Définir les arguments de ligne de commande
	flag.StringVar(&port, "port", "8078", "Port d'écoute du pair")
	flag.StringVar(&fileName, "file", "", "Nom du fichier à publier ou rechercher")
	flag.Parse()

	// Si un fichier est spécifié, démarrer le serveur pour publier le fichier
	if fileName != "" {
		go runServer(port, fileName)
		select {}
	}

	// Sinon, démarrer le serveur sans publier de fichier
	go runServer(port, "")

	// Connexion au serveur index (remplacez "localhost:8077" par l'adresse correcte)
	serverConn, err := net.Dial("tcp", "localhost:8077")
	if err != nil {
		fmt.Println("Erreur lors de la connexion au serveur index:", err)
		return
	}
	defer serverConn.Close()

	// Menu pour choisir l'option
	var option int
	fmt.Println("1. Publier un fichier")
	fmt.Println("2. Rechercher un fichier")
	fmt.Println("3. Demander un fichier à un autre pair")
	fmt.Print("Choisissez une option: ")
	fmt.Scanln(&option)

	switch option {
	case 1:
		fmt.Print("Entrez le nom du fichier à publier: ")
		fmt.Scanln(&fileName)

		_, err := serverConn.Write([]byte("publish\n"))
		if err != nil {
			fmt.Println("Erreur lors de l'envoi de la demande de publication:", err)
			return
		}

		clientInfo := fmt.Sprintf("localhost:%s;%s\n", port, fileName)
		_, err = serverConn.Write([]byte(clientInfo))
		if err != nil {
			fmt.Println("Erreur lors de l'envoi du nom du fichier publié:", err)
			return
		}

		response := make([]byte, 1024)
		n, err := serverConn.Read(response)
		if err != nil {
			fmt.Println("Erreur lors de la réception de la réponse du serveur:", err)
			return
		}
		fmt.Println("Réponse du serveur:", string(response[:n]))

	case 2:
		fmt.Print("Entrez le nom du fichier à rechercher: ")
		fmt.Scanln(&fileName)

		_, err := serverConn.Write([]byte("search\n"))
		if err != nil {
			fmt.Println("Erreur lors de l'envoi de la demande de recherche:", err)
			return
		}

		_, err = serverConn.Write([]byte(fileName + "\n"))
		if err != nil {
			fmt.Println("Erreur lors de l'envoi du nom du fichier à rechercher:", err)
			return
		}

		var peerIP [1024]byte
		n, err := serverConn.Read(peerIP[:])
		if err != nil {
			fmt.Println("Erreur lors de la réception de l'adresse IP du pair possédant le fichier:", err)
			return
		}
		ip := strings.TrimSpace(string(peerIP[:n]))

		if ip == "not_found" {
			fmt.Println("Le fichier recherché n'existe pas.")
			return
		}
		fmt.Println("Adresse IP du pair possédant le fichier:", ip)

	case 3:
		fmt.Print("Entrez le nom du fichier à demander: ")
		fmt.Scanln(&fileName)

		_, err := serverConn.Write([]byte("request\n"))
		if err != nil {
			fmt.Println("Erreur lors de l'envoi de la demande de fichier:", err)
			return
		}

		_, err = serverConn.Write([]byte(fileName + "\n"))
		if err != nil {
			fmt.Println("Erreur lors de l'envoi du nom du fichier demandé:", err)
			return
		}

		var peerIPPort [1024]byte
		n, err := serverConn.Read(peerIPPort[:])
		if err != nil {
			fmt.Println("Erreur lors de la réception de l'adresse IP et du port du pair possédant le fichier:", err)
			return
		}
		ipPort := strings.TrimSpace(string(peerIPPort[:n]))

		if ipPort == "not_found" {
			fmt.Println("Le fichier demandé n'existe pas.")
			return
		}
		fmt.Println("Adresse IP et port du pair possédant le fichier:", ipPort)

		// Séparer l'adresse IP et le port
		parts := strings.Split(ipPort, ":")
		if len(parts) != 2 {
			fmt.Println("Adresse IP et port non valides.")
			return
		}
		ip := parts[0]
		port := parts[1]

		runClient(ip, port, fileName)

	default:
		fmt.Println("Option non valide.")
	}
}

func runClient(ip, port, fileName string) {
	conn, err := net.Dial("tcp", ip+":"+port)
	if err != nil {
		fmt.Println("Erreur lors de la connexion:", err)
		return
	}
	defer conn.Close()

	fmt.Println("Connecté au pair.", ip, port)

	// Envoyer le nom du fichier
	_, err = conn.Write([]byte(fileName + "\n"))
	if err != nil {
		fmt.Println("Erreur lors de l'envoi du nom du fichier:", err)
		return
	}

	receiveFile(conn)
}

func handleConnection(conn net.Conn, fileName string) {
	defer conn.Close()

	fmt.Println("Connexion acceptée.")
	// Recevoir le nom du fichier demandé
	var requestedFileNameBuffer [1024]byte
	n, err := conn.Read(requestedFileNameBuffer[:])
	if err != nil {
		fmt.Println("Erreur lors de la lecture du nom du fichier demandé:", err)
		return
	}
	requestedFileName := strings.TrimSpace(string(requestedFileNameBuffer[:n]))

	// Envoyer le fichier demandé
	sendFile(conn, requestedFileName)
}

func sendFile(conn net.Conn, fileName string) {
	// Envoi du nom du fichier
	_, err := conn.Write([]byte(fileName))
	if err != nil {
		fmt.Println("Erreur lors de l'envoi du nom du fichier:", err)
		return
	}

	// Ouverture du fichier
	file, err := os.Open(fileName)
	if err != nil {
		fmt.Println("Erreur lors de l'ouverture du fichier:", err)
		return
	}
	defer file.Close()

	// Envoi du contenu du fichier
	_, err = io.Copy(conn, file)
	if err != nil {
		fmt.Println("Erreur lors de l'envoi du fichier:", err)
		return
	}

	fmt.Printf("Fichier \"%s\" envoyé avec succès.\n", fileName)
}

func receiveFile(conn net.Conn) {
	// Réception du nom du fichier
	var receivedFileNameBuffer [1024]byte
	n, err := conn.Read(receivedFileNameBuffer[:])
	if err != nil {
		fmt.Println("Erreur lors de la lecture du nom du fichier:", err)
		return
	}
	receivedFileName := strings.TrimSpace(string(receivedFileNameBuffer[:n]))

	// Ouverture du fichier pour écriture
	file, err := os.Create(receivedFileName)
	if err != nil {
		fmt.Println("Erreur lors de la création du fichier:", err)
		return
	}
	defer file.Close()

	// Réception et écriture du contenu du fichier
	_, err = io.Copy(file, conn)
	if err != nil {
		fmt.Println("Erreur lors de la réception du fichier:", err)
		return
	}

	fmt.Printf("Fichier \"%s\" reçu avec succès.\n", receivedFileName)
}

func runServer(port, fileName string) {
	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		fmt.Println("Erreur lors de l'écoute:", err)
		return
	}
	defer listener.Close()

	fmt.Printf("En attente de connexion sur le port %s...\n", port)

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Erreur lors de l'acceptation de la connexion:", err)
			continue
		}

		go handleConnection(conn, fileName)
	}
}
