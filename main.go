package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	defaultFileDownloadTimeout = 20 * time.Second // Duração que o coletor deve esperar até que o download de cada um dos arquivos seja concluído
	defaultGeneralTimeout      = 6 * time.Minute  // Duração máxima total da coleta de todos os arquivos. Valor padrão calculado a partir de uma média de execuções ~4.5min
	defaulTimeBetweenSteps     = 5 * time.Second  //Tempo de espera entre passos do coletor."
)

func main() {
	if _, err := strconv.Atoi(os.Getenv("MONTH")); err != nil {
		log.Fatalf("Invalid month (\"%s\"): %q", os.Getenv("MONTH"), err)
	}
	month := os.Getenv("MONTH")

	if _, err := strconv.Atoi(os.Getenv("YEAR")); err != nil {
		log.Fatalf("Invalid year (\"%s\"): %q", os.Getenv("YEAR"), err)
	}
	year := os.Getenv("YEAR")

	outputFolder := os.Getenv("OUTPUT_FOLDER")
	if outputFolder == "" {
		outputFolder = "/output"
	}

	if err := os.Mkdir(outputFolder, os.ModePerm); err != nil && !os.IsExist(err) {
		log.Fatalf("Error creating output folder(%s): %q", outputFolder, err)
	}

	monthMap := map[string]string{
		"01": "Janeiro",
		"02": "Fevereiro",
		"03": "Marco",
		"04": "Abril",
		"05": "Maio",
		"06": "Junho",
		"07": "Julho",
		"08": "Agosto",
		"09": "Setembro",
		"10": "Outubro",
		"11": "Novembro",
		"12": "Dezembro",
	}
	cLink := fmt.Sprintf("http://www.transparencia.mpf.mp.br/conteudo/contracheque/remuneracao-membros-ativos/%s/remuneracao-membros-ativos_%s_%s.ods", year, year, monthMap[month])
	cPath := filepath.Join(outputFolder, fmt.Sprintf("membros-ativos-contracheques-%s-%s.ods", month, year))
	log.Printf("Baixando arquivo %s\n", cLink)
	if err := download(cLink, cPath); err != nil {
		log.Fatal(err)
	}
	log.Printf("Arquivo baixado com sucesso!\n")

	iLink := fmt.Sprintf("http://www.transparencia.mpf.mp.br/conteudo/contracheque/verbas-indenizatorias-e-outras-remuneracoes-temporarias/membros-ativos/%s/verbas-indenizatorias-e-outras-remuneracoes-temporarias_%s_%s.ods", year, year, monthMap[month])
	iPath := filepath.Join(outputFolder, fmt.Sprintf("membros-ativos-indenizacoes-%s-%s.ods", month, year))
	log.Printf("Baixando arquivo %s\n", iLink)
	if err := download(iLink, iPath); err != nil {
		log.Fatal(err)
	}
	log.Printf("Arquivo baixado com sucesso!\n")
	// O parser do MPF espera os arquivos separados por \n. Mudanças aqui tem que
	// refletir as expectativas lá.
	fmt.Println(strings.Join([]string{cPath, iPath}, "\n"))
}

func download(url, path string) error {
	resp, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	cFile, err := os.Create(path)
	if err != nil {
		log.Fatal(err)
	}
	defer cFile.Close()
	cWriter := bufio.NewWriter(cFile)
	if _, err := io.Copy(cWriter, resp.Body); err != nil {
		log.Fatal(err)
	}
	cWriter.Flush()
	return nil
}
