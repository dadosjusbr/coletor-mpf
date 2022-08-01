package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/chromedp/cdproto/browser"
	"github.com/chromedp/chromedp"
)

type crawler struct {
	// Aqui temos os atributos e métodos necessários para realizar a coleta dos dados
	downloadTimeout  time.Duration
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

	//Criando o contexto do chromedp
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
	if err := c.selecionaAno(ctx, "contra", c.year); err != nil {
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
	monthConverted, err := strconv.Atoi(c.month)
	if err != nil {
		log.Fatal("erro ao converter mês para inteiro")
	}
	yearConverted, err := strconv.Atoi(c.year)
	if err != nil {
		log.Fatal("erro ao converter ano para inteiro")
	}
	// A publicação dos relatórios de Verbas Indenizatórias e outras Remunerações Temporárias
	// foi iniciada no mês de julho de 2019, em função do início da vigência da Resolução CNMP Nº 200
	if yearConverted > 2019 || yearConverted == 2019 && monthConverted >= 7 {
		log.Printf("Realizando seleção (%s/%s)...", c.month, c.year)
		if err := c.selecionaAno(ctx, "inde", c.year); err != nil {
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
	return []string{cqFname}, nil
}

// Retorna os caminhos completos dos arquivos baixados.
func (c crawler) downloadFilePath(prefix string) string {
	return filepath.Join(c.output, fmt.Sprintf("membros-ativos-%s-%s-%s.ods", prefix, c.month, c.year))
}

func (c crawler) selecionaAno(ctx context.Context, tipo string, year string) error {
	var baseURL string

	if tipo == "contra" {
		baseURL = "http://www.transparencia.mpf.mp.br/conteudo/contracheque/remuneracao-membros-ativos"
		return chromedp.Run(ctx,
			chromedp.Navigate(baseURL),
			chromedp.Sleep(c.timeBetweenSteps),

			// Seleciona o ano
			chromedp.SetValue(`//*[@id="select_ano"]`, year, chromedp.BySearch),
			chromedp.Sleep(c.timeBetweenSteps),

			// Altera o diretório de download
			browser.SetDownloadBehavior(browser.SetDownloadBehaviorBehaviorAllowAndName).
				WithDownloadPath(c.output).
				WithEventsEnabled(true),
		)
	} else {
		baseURL = "http://www.transparencia.mpf.mp.br/conteudo/contracheque/verbas-indenizatorias-e-outras-remuneracoes-temporarias"
		return chromedp.Run(ctx,
			chromedp.Navigate(baseURL),
			chromedp.Sleep(c.timeBetweenSteps),

			// Seleciona a opção -> Membros Ativos
			chromedp.SetValue(`//*[@id="select_opcao1"]`, "membros-ativos", chromedp.BySearch, chromedp.NodeVisible),
			chromedp.Sleep(c.timeBetweenSteps),

			// Seleciona o ano
			chromedp.SetValue(`//*[@id="selectAno"]`, year, chromedp.BySearch, chromedp.NodeVisible),
			chromedp.Sleep(c.timeBetweenSteps),

			// Consulta
			chromedp.DoubleClick(`//*[@id="btnConsultar"]`, chromedp.BySearch, chromedp.NodeVisible),
			chromedp.Sleep(c.timeBetweenSteps),

			// Altera o diretório de download
			browser.SetDownloadBehavior(browser.SetDownloadBehaviorBehaviorAllowAndName).
				WithDownloadPath(c.output).
				WithEventsEnabled(true),
		)
	}
}

// A função exportaPlanilha clica no botão correto para exportar para excel, espera um tempo para o download e renomeia o arquivo.
func (c crawler) exportaPlanilha(ctx context.Context, fName string) error {
	var link string
	monthConverted, err := strconv.Atoi(c.month)
	if err != nil {
		log.Fatal("erro ao converter mês para inteiro")
	}
	if monthConverted <= 6 {
		link = fmt.Sprintf(`/html/body/div[2]/div[2]/div[1]/div[1]/div/div[3]/div/div[4]/div[2]/div[1]/div[%d]/a`, 3+5*(monthConverted-1))
	} else {
		link = fmt.Sprintf(`/html/body/div[2]/div[2]/div[1]/div[1]/div/div[3]/div/div[4]/div[2]/div[2]/div[%d]/a`, 3+5*(monthConverted-1))
	}

	chromedp.Run(ctx,
		// Clica no botão de download
		chromedp.DoubleClick(link, chromedp.BySearch, chromedp.NodeVisible),
		chromedp.Sleep(c.timeBetweenSteps),
	)
	time.Sleep(c.downloadTimeout)
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
