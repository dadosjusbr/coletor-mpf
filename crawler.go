package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/chromedp/cdproto/browser"
	"github.com/chromedp/chromedp"
)

type crawler struct {
	// Aqui temos os atributos e métodos necessários para realizar a coleta dos dados
	generalTimeout   time.Duration
	timeBetweenSteps time.Duration
	year             string
	month            string
	output           string
}

func (c crawler) crawl() ([]string, error) {
	// Chromedp setup.
	log.SetOutput(os.Stderr) // Enviando logs para o stderr para não afetar a execução do coletor.
	alloc, allocCancel := chromedp.NewExecAllocator(
		context.Background(),
		append(chromedp.DefaultExecAllocatorOptions[:],
			chromedp.UserAgent("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_14_5) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/77.0.3830.0 Safari/537.36"),
			chromedp.Flag("headless", true), // mude para false para executar com navegador visível.
			chromedp.NoSandbox,
			chromedp.DisableGPU,
		)...,
	)
	defer allocCancel()

	ctx, cancel := chromedp.NewContext(
		alloc,
		chromedp.WithLogf(log.Printf), // remover comentário para depurar
	)
	defer cancel()

	ctx, cancel = context.WithTimeout(ctx, c.generalTimeout)
	defer cancel()

	// NOTA IMPORTANTE: os prefixos dos nomes dos arquivos tem que ser igual
	// ao esperado no parser MPF.

	// Realiza o download
	// Contracheque
	log.Printf("Realizando seleção (%s/%s)...", c.month, c.year)
	if err := c.selecionaAnoMes(ctx, "contra", c.year, c.month); err != nil {
		log.Fatalf("Erro no setup:%v", err)
	}
	log.Printf("Seleção realizada com sucesso!\n")
	cqFname := c.downloadFilePath("contracheque")
	log.Printf("Fazendo download do contracheque (%s)...", cqFname)
	if err := c.exportaPlanilha(ctx, cqFname); err != nil {
		log.Fatalf("Erro fazendo download do contracheque: %v", err)
	}
	log.Printf("Download realizado com sucesso!\n")

	// Indenizações
	log.Printf("Realizando seleção (%s/%s)...", c.month, c.year)
	if err := c.selecionaAnoMes(ctx, "inde", c.year, c.month); err != nil {
		log.Fatalf("Erro no setup:%v", err)
	}
	log.Printf("Seleção realizada com sucesso!\n")
	iFname := c.downloadFilePath("indenizatorias")
	log.Printf("Fazendo download das indenizações (%s)...", iFname)
	if err := c.exportaPlanilha(ctx, iFname); err != nil {
		log.Fatalf("Erro fazendo download dos indenizações: %v", err)
	}
	log.Printf("Download realizado com sucesso!\n")

	return []string{cqFname, iFname}, nil
}

// Retorna os caminhos completos dos arquivos baixados.
func (c crawler) downloadFilePath(prefix string) string {
	return filepath.Join(c.output, fmt.Sprintf("membros-ativos-%s-%s-%s.ods", prefix, c.month, c.year))
}

func (c crawler) selecionaAnoMes(ctx context.Context, tipo string, year string, month string) error {
	var baseURL string
	//selectYear := `//*[@id="select_ano"]`

	//title := fmt.Sprintf(`//*[@title="Remuneração de Membros Ativos - %s - ODS"]`, selectMonth[month])
	if tipo == "contra" {
		baseURL = "http://www.transparencia.mpf.mp.br/conteudo/contracheque/remuneracao-membros-ativos"
	} else {
		baseURL = "http://www.transparencia.mpf.mp.br/conteudo/contracheque/verbas-indenizatorias-e-outras-remuneracoes-temporarias"
	}

	return chromedp.Run(ctx,
		chromedp.Navigate(baseURL),
		chromedp.Sleep(c.timeBetweenSteps),

		// Seleciona o ano
		/*chromedp.SetValue(selectYear, year, chromedp.BySearch),
		chromedp.Sleep(c.timeBetweenSteps),*/

		// Altera o diretório de download
		browser.SetDownloadBehavior(browser.SetDownloadBehaviorBehaviorAllowAndName).
			WithDownloadPath(c.output).
			WithEventsEnabled(true),
	)
}

// A função exportaPlanilha clica no botão correto para exportar para excel, espera um tempo para o download e renomeia o arquivo.
func (c crawler) exportaPlanilha(ctx context.Context, fName string) error {
	var link string
	var baseURL string
	selectMonth := map[string]string{"01": "Janeiro", "02": "Fevereiro", "03": "Março", "04": "Abril", "05": "Maio", "06": "Junho", "07": "Julho", "08": "Agosto", "09": "Setembro", "10": "Outubro", "11": "Novembro", "12": "Dezembro"}
	if strings.Contains(fName, "contracheque") {
		baseURL = "http://www.transparencia.mpf.mp.br/conteudo/contracheque/remuneracao-membros-ativos"
		link = fmt.Sprintf("/%s/remuneracao-membros-ativos_%s_%s.ods", c.year, c.year, selectMonth[c.month])
		chromedp.Run(ctx,
			// Clica no botão de download
			chromedp.Navigate(baseURL+link),
			chromedp.Sleep(c.timeBetweenSteps),
		)
	} else {
		baseURL = "http://www.transparencia.mpf.mp.br/conteudo/contracheque/verbas-indenizatorias-e-outras-remuneracoes-temporarias"
		link = fmt.Sprintf("/membros-ativos/%s/verbas-indenizatorias-e-outras-remuneracoes-temporarias_%s_%s.ods", c.year, c.year, selectMonth[c.month])
		chromedp.Run(ctx,
			// Clica no botão de download
			chromedp.Navigate(baseURL+link),
			chromedp.Sleep(c.timeBetweenSteps),
		)
	}

	if err := nomeiaDownload(c.output, fName); err != nil {
		return fmt.Errorf("erro renomeando arquivo (%s): %v", fName, err)
	}
	if _, err := os.Stat(fName); os.IsNotExist(err) {
		return fmt.Errorf("download do arquivo de %s não realizado", fName)
	}
	return nil
}

// A função nomeiaDownload dá um nome ao último arquivo modificado dentro do diretório
// passado como parâmetro
func nomeiaDownload(output, fName string) error {
	// Identifica qual foi o último arquivo
	files, err := os.ReadDir(output)
	if err != nil {
		return fmt.Errorf("erro lendo diretório %s: %v", output, err)
	}
	var newestFPath string
	var newestTime int64 = 0
	for _, f := range files {
		fPath := filepath.Join(output, f.Name())
		fi, err := os.Stat(fPath)
		if err != nil {
			return fmt.Errorf("erro obtendo informações sobre arquivo %s: %v", fPath, err)
		}
		currTime := fi.ModTime().Unix()
		if currTime > newestTime {
			newestTime = currTime
			newestFPath = fPath
		}
	}
	// Renomeia o último arquivo modificado.
	if err := os.Rename(newestFPath, fName); err != nil {
		return fmt.Errorf("erro renomeando último arquivo modificado (%s)->(%s): %v", newestFPath, fName, err)
	}
	return nil
}
