# Como exportar dados do Lens.org

O Lens.org não oferece API gratuita para downloads em lote. É necessário fazer o export manual.

## Passo a passo

### 1. Acesse o Lens.org

Abra: https://www.lens.org/lens/search/patent/list

### 2. Execute a busca por patentes da UFV

Na caixa de busca, cole a query:

```
applicant.name:("UNIVERSIDADE FEDERAL DE VICOSA" OR "UNIVERSIDADE FEDERAL DE VIÇOSA" OR "UFV")
```

Marque o filtro **Jurisdiction: BR** para focar em patentes brasileiras.

### 3. Exporte o CSV

1. Clique no botão **Export** (ícone de seta para baixo, no canto superior direito dos resultados)
2. Selecione **CSV**
3. Selecione **All columns** (importante: precisamos de todas as colunas para citações e família)
4. Limite: 1000 registros por export. Se houver mais, exporte em lotes filtrando por faixa de ano.
5. Salve o arquivo como `lens_export.csv`

### 4. Mova o arquivo para o local correto

```bash
mv ~/Downloads/lens_export.csv ~/Downloads/lens_export.csv
```

Ou informe o caminho diretamente:

```bash
make collect-lens INPUT=~/Downloads/lens_export.csv
```

### 5. Execute o parser

```bash
make collect-lens
make ingest-lens
```

## Colunas que o parser usa

| Coluna Lens | Campo interno |
|---|---|
| Lens ID | `lens_id` |
| Title | `title` |
| Abstract | `abstract` |
| Application Number | `application_number` (usado como `inpi_number` se começar com BR) |
| Filing Date | `filing_date` |
| Publication Date | `publication_date` |
| Grant Date | `grant_date` |
| Applicants | `applicants[]` |
| Inventors | `inventors[]` |
| Jurisdiction | `jurisdiction` |
| Legal Status | `legal_status` |
| International Classifications | `ipc_codes[]` |
| Patent Citations | `patent_citations[]` |
| Non Patent Citations | `npl_citations[]` |
| Cited By Patent Count | `cited_by_count` |
| Simple Family Size | `family_size` |

## Exportes em lote (> 1000 patentes)

Se encontrar mais de 1000 resultados:

1. Filtre por ano: ex. 1990–2000, 2001–2010, 2011–2020, 2021–presente
2. Exporte cada lote como `lens_export_1990_2000.csv`, etc.
3. Execute o parser para cada arquivo:
   ```bash
   python3 ai-service/collectors/lens_parser.py --input ~/Downloads/lens_export_1990_2000.csv
   python3 ai-service/collectors/lens_parser.py --input ~/Downloads/lens_export_2001_2010.csv
   ```
   O parser faz append no mesmo JSONL — pode rodar múltiplas vezes.

## Observações

- O Lens.org atualiza os dados semanalmente. Recomenda-se re-exportar mensalmente.
- Patentes em família internacional aparecem com `family_size > 1`.
- O campo `npl_citations` contém referências a artigos científicos — usado para ligar patentes UFV a publicações no banco Ágora.
