-- Tabela de matches entre grupos de pesquisa e parceiros potenciais
CREATE TABLE IF NOT EXISTS matches (
    id           BIGSERIAL PRIMARY KEY,
    group_id     BIGINT REFERENCES research_groups(id) ON DELETE CASCADE,
    partner_id   BIGINT REFERENCES partners(id) ON DELETE CASCADE,
    score        NUMERIC(5,4) NOT NULL DEFAULT 0,
    reasons      JSONB,
    status       TEXT NOT NULL DEFAULT 'pending',  -- pending | contacted | in_progress | closed
    created_at   TIMESTAMPTZ DEFAULT NOW(),
    updated_at   TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE (group_id, partner_id)
);
CREATE INDEX IF NOT EXISTS idx_matches_score  ON matches(score DESC);
CREATE INDEX IF NOT EXISTS idx_matches_status ON matches(status);
CREATE INDEX IF NOT EXISTS idx_matches_group  ON matches(group_id);

-- Seed: empresas brasileiras relevantes para as áreas de pesquisa da UFV
-- normalized_name = lowercase sem acentos/especiais; gerado via expressão SQL
INSERT INTO partners (name, normalized_name, partner_type, sector, location, cnae_code, interest_score, source)
VALUES
  -- Defensivos / Agroquímicos
  ('Bayer CropScience Brasil',          lower('bayer cropscience brasil'),             'empresa',         'Defensivos Agrícolas',         'São Paulo / SP',         '2091-6/00', 0.90, 'seed'),
  ('Syngenta Proteção de Cultivos',     lower('syngenta protecao de cultivos'),        'empresa',         'Defensivos Agrícolas',         'São Paulo / SP',         '2091-6/00', 0.90, 'seed'),
  ('BASF Agricultural Solutions',       lower('basf agricultural solutions'),          'empresa',         'Defensivos Agrícolas',         'São Paulo / SP',         '2091-6/00', 0.85, 'seed'),
  ('Corteva Agriscience Brasil',        lower('corteva agriscience brasil'),           'empresa',         'Sementes e Defensivos',        'São Paulo / SP',         '2091-6/00', 0.85, 'seed'),
  ('FMC Química do Brasil',             lower('fmc quimica do brasil'),                'empresa',         'Defensivos Agrícolas',         'Campinas / SP',          '2091-6/00', 0.80, 'seed'),
  ('UPL do Brasil',                     lower('upl do brasil'),                        'empresa',         'Defensivos Agrícolas',         'Ribeirão Preto / SP',    '2091-6/00', 0.80, 'seed'),
  ('Mosaic Fertilizantes',              lower('mosaic fertilizantes'),                 'empresa',         'Fertilizantes',                'Uberaba / MG',           '2012-6/00', 0.85, 'seed'),
  ('Heringer Fertilizantes',            lower('heringer fertilizantes'),               'empresa',         'Fertilizantes',                'Uberaba / MG',           '2012-6/00', 0.80, 'seed'),
  ('Yara Brasil Fertilizantes',         lower('yara brasil fertilizantes'),            'empresa',         'Fertilizantes',                'São Paulo / SP',         '2012-6/00', 0.80, 'seed'),
  ('Koppert Biological Systems',        lower('koppert biological systems'),           'empresa',         'Controle Biológico',           'Piracicaba / SP',        '7210-0/00', 0.95, 'seed'),
  ('Promip Manejo Integrado de Pragas', lower('promip manejo integrado de pragas'),    'empresa',         'Controle Biológico',           'Engenheiro Coelho / SP', '7210-0/00', 0.90, 'seed'),
  ('Biobest Brasil',                    lower('biobest brasil'),                       'empresa',         'Controle Biológico',           'São Paulo / SP',         '7210-0/00', 0.88, 'seed'),
  ('Nestlé Brasil',                     lower('nestle brasil'),                        'empresa',         'Alimentos e Bebidas',          'Araras / SP',            '1099-6/99', 0.85, 'seed'),
  ('BRF S.A.',                          lower('brf sa'),                               'empresa',         'Alimentos Processados',        'São Paulo / SP',         '1013-9/01', 0.85, 'seed'),
  ('JBS S.A.',                          lower('jbs sa'),                               'empresa',         'Frigorífico e Proteínas',      'São Paulo / SP',         '1011-2/01', 0.80, 'seed'),
  ('Lactalis do Brasil',                lower('lactalis do brasil'),                   'empresa',         'Laticínios',                   'São Paulo / SP',         '1052-0/00', 0.80, 'seed'),
  ('Vigor Alimentos',                   lower('vigor alimentos'),                      'empresa',         'Laticínios',                   'São Paulo / SP',         '1052-0/00', 0.75, 'seed'),
  ('Suzano Papel e Celulose',           lower('suzano papel e celulose'),              'empresa',         'Florestal e Celulose',         'São Paulo / SP',         '1721-4/00', 0.90, 'seed'),
  ('Klabin S.A.',                       lower('klabin sa'),                            'empresa',         'Florestal e Celulose',         'São Paulo / SP',         '1721-4/00', 0.88, 'seed'),
  ('Dexco (Duratex)',                   lower('dexco duratex'),                        'empresa',         'Florestal e Madeira',          'São Paulo / SP',         '1622-6/02', 0.80, 'seed'),
  ('EMS Farmacêutica',                  lower('ems farmaceutica'),                     'empresa',         'Farmacêutica e Biotecnologia', 'Hortolândia / SP',       '2121-1/01', 0.85, 'seed'),
  ('Eurofarma Laboratórios',            lower('eurofarma laboratorios'),               'empresa',         'Farmacêutica',                 'São Paulo / SP',         '2121-1/01', 0.82, 'seed'),
  ('Cristália Produtos Químicos',       lower('cristalia produtos quimicos'),          'empresa',         'Farmacêutica',                 'Itapira / SP',           '2121-1/01', 0.80, 'seed'),
  ('Biomm S.A.',                        lower('biomm sa'),                             'empresa',         'Biotecnologia',                'Nova Lima / MG',         '7210-0/00', 0.88, 'seed'),
  ('John Deere Brasil',                 lower('john deere brasil'),                    'empresa',         'Máquinas Agrícolas',           'Horizontina / RS',       '2833-0/00', 0.85, 'seed'),
  ('CNH Industrial Brasil',             lower('cnh industrial brasil'),                'empresa',         'Máquinas Agrícolas',           'Curitiba / PR',          '2833-0/00', 0.83, 'seed'),
  ('AGCO do Brasil',                    lower('agco do brasil'),                       'empresa',         'Máquinas Agrícolas',           'Santa Rosa / RS',        '2833-0/00', 0.82, 'seed'),
  ('TOTVS S.A.',                        lower('totvs sa'),                             'empresa',         'Software de Gestão',           'São Paulo / SP',         '6201-5/01', 0.75, 'seed'),
  ('Agrotools',                         lower('agrotools'),                            'empresa',         'Agritech e Dados',             'São Paulo / SP',         '7210-0/00', 0.85, 'seed'),
  ('Solinftec',                         lower('solinftec'),                            'empresa',         'Agritech e IA',                'Araçatuba / SP',         '7210-0/00', 0.88, 'seed'),
  ('Embrapa',                           lower('embrapa'),                              'empresa',         'Pesquisa Agropecuária',        'Brasília / DF',          '7210-0/00', 0.95, 'seed'),
  ('Fapemig',                           lower('fapemig'),                              'orgao_fomento',   'Fomento à Pesquisa',           'Belo Horizonte / MG',    '9411-1/00', 0.90, 'seed')
ON CONFLICT (normalized_name) DO NOTHING;
