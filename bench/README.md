# Releve de vitesse du decodeur de film

Cette commande mesure la vitesse du decodeur seul, version par version, sur un jeu fixe de 13 films.
Le releve d'une version se range dans `bench/releves/vX.Y.Z.tsv` ; comparer deux versions revient a
comparer deux fichiers.

**Ce releve N'EST PAS le temps d'une generation complete de rejeu.** Il tourne avec des options
minimales (la carte seule) : ni source de degat, ni tableau, ni objectifs, ni zones. Une generation de
rejeu complete lit aussi une base de donnees, absente de ce depot : elle se mesure ailleurs (releve B,
cote application). Les durees d'ici sont donc plus courtes que celles d'une generation.

## Lancer

```
GOWORK=off go run ./cmd/film-bench \
  -films bench/films.tsv \
  -correspondance <fichier local hors depot> \
  -catalogue <map_quant_bounds.json> \
  -passes 5 \
  -sortie bench/releves/vX.Y.Z.tsv
```

| Option | Sens |
|---|---|
| `-films` | jeu fixe (defaut `bench/films.tsv`) |
| `-correspondance` | fichier local, voir plus bas (obligatoire) |
| `-catalogue` | catalogue des bornes de carte du jeu (obligatoire) |
| `-passes` | passes chronometrees, defaut 5, apres une passe d'echauffement ecartee |
| `-sortie` | releve TSV a ecrire ; sans cette option, seul le resume s'affiche ; un chemin sous un dossier `data` est refuse |
| `-seulement` | noms de films a mesurer, separes par des virgules (par exemple `film05,film02`) |
| `-plafond-gio` | plafond de memoire d'un film pour la mesure du pic, defaut 2 Gio ; un film qui le franchit arrete le releve |
| `-fils` | nombre de fils que le decodeur peut employer, defaut 1 (comportement actuel) ; le nombre employe est plafonne par `GOMAXPROCS` et par `replay.MaxFils`, et c'est lui que la colonne `fils` porte ; s'applique a toutes les passes, y compris celle du pic |
| `-etapes` | fichier TSV de la suite ordonnee des etapes observees (voir plus bas) ; un chemin sous un dossier `data` est refuse ; incompatible avec `-enfant` et les profils |

L'outil n'emploie que des entrees publiques du decodeur. Les options `-fils` et `-etapes` s'appuient sur
`replay.Options.Vitesse`, qui n'existe qu'a partir de la version qui la porte : pour relever une version
anterieure, copier l'outil dans son arbre apres avoir retire ces deux options.

## Le jeu fixe et la correspondance

`bench/films.tsv` ne porte que des noms neutres (`film01` a `film13`), le mode, la version, le nombre
de joueurs et le nombre de morceaux. Il ne contient aucun fichier du jeu, aucun morceau, aucun
identifiant de film ou de joueur.

Les vrais films se retrouvent par un fichier LOCAL, jamais range au depot, donne par
`-correspondance`. Une ligne par film, quatre colonnes separees par des tabulations (les lignes vides
et celles qui commencent par `#` sont ignorees) :

```
film01	<dossier des morceaux>	<manifeste du film>	<carte>
```

- le dossier des morceaux porte les `chunk_NN.bin` du film ; le manifeste est son fichier JSON ;
- la carte est le nom de la carte ou son module, retrouve au catalogue de bornes (la carte ne se
  deduit pas du film : elle vient de l'enregistrement du match) ;
- un chemin relatif se lit depuis le dossier du fichier de correspondance ;
- un dossier ou un manifeste manquant, un film du jeu absent de la correspondance, ou un nombre de
  morceaux different de celui du jeu fixe arrete l'outil.

### Criteres du jeu de 13 films

Pris dans cet ordre parmi les films presents sur le poste de mesure : couverture des modes (Slayer,
Grande bataille, Drapeau, Bases, Colline, Crane, autres), des versions de format (31 a 41) et des
tailles (de 13 a 69 morceaux) ; les deux temoins de duree conserves pour la continuite des mesures ;
un film de Grande bataille de la build la plus recente de son type ; au moins un film a plus de 24
joueurs (cinq en ont) ; la borne basse (le plus petit film) et la borne haute (le plus gros). Le film
de plus fort pic de memoire du parc n'est pas designe par mesure : le plus gros film est retenu comme
candidat. Aucun film de plus de 30 joueurs n'est present.

Le jeu public (petits films du depot, executable sans donnee de l'hote) n'est pas fourni par cet outil.

## Ce que mesure chaque phase

Toutes les durees sont en millisecondes. Une ligne du releve par film et par phase.

| Phase | Mesure | Entree publique |
|---|---|---|
| `decompression` | somme de la decompression de chaque morceau, octets lus avant le chronometre | `decfilm.Inflate` |
| `chargement` | lecture des fichiers, decompression et decoupage en paquets, ensemble | `decfilm.LoadDir` |
| `decodage` | du film aux faits : du debut de la construction a l'etape `clockOrigin` | `replay.BuildFromFilmAvecFaits`, observateur d'etapes (`Options.Observe`) |
| `publication` | des faits au document : de l'etape `clockOrigin` au retour de l'appel | idem |
| `total` | chargement, decodage et publication d'un seul tenant (une seule horloge) | idem |

- La `decompression` est CONTENUE dans le `chargement` : elle n'est pas ajoutee au total. Elle donne la
  part de la decompression, qui est la cible du travail sur plusieurs fils.
- Dans le `chargement`, la lecture des fichiers et le decoupage en paquets ne se separent pas par
  l'entree publique : aucune estimation n'en est faite. La lecture est faite sur des fichiers deja en
  cache du systeme (l'echauffement et la lecture prealable de la decompression les y mettent).
- `total` = `chargement` + `decodage` + `publication`, a l'ecart de quelques microsecondes pres.

Les colonnes du TSV :

| Colonne | Sens |
|---|---|
| `film`, `mode`, `version`, `joueurs`, `morceaux` | le film, comme dans `bench/films.tsv` |
| `fils` | nombre de fils employes par le decodeur pour ce releve (`-fils`, plafonne par `GOMAXPROCS`) ; 1 pour une version sans option de fils |
| `phase` | une des cinq phases ci-dessus |
| `mediane_ms`, `min_ms`, `max_ms`, `passes` | mediane, minimum et maximum de la phase sur les passes chronometrees, et leur nombre |
| `pic_mio` | pic de memoire en Mio, pris dans un processus neuf (voir plus bas) ; repete sur chaque ligne du film |
| `empreinte_document`, `empreinte_faits` | SHA-256 du document de rejeu (serialise en JSON) et du fichier de faits ; repetees sur chaque ligne du film |

Les empreintes servent a prouver qu'une optimisation ne change aucune sortie : deux releves du meme
code donnent les memes empreintes, et une version qui ne doit rien changer les garde.

L'en-tete commente du fichier dit la date, l'arbre mesure (`git describe`), la version de Go, le
systeme, le processeur, le nombre de coeurs, la memoire, `GOMAXPROCS`, le nombre de passes et le
plafond de memoire. Il ne contient aucun chemin ni nom de poste.

## Protocole

1. **Fichiers lus avant le chronometre.** Les octets des morceaux sont lus en memoire avant la mesure
   de la decompression, et une passe d'echauffement ecartee precede les passes retenues : le disque
   n'entre pas dans les durees.
2. **Un processus neuf par film pour le pic de memoire.** L'outil se relance lui-meme, un film a la
   fois, et ce fils fait une passe sous la sentinelle de memoire du decodeur (`memguard`, plafond de
   mesure de 2 Gio). Le pic est l'empreinte memoire du tas Go (memoire obtenue du systeme moins la
   memoire rendue), pas la memoire residente du processus. Il est lu avant le calcul des empreintes.
3. **Echauffement, puis `-passes` passes (5 par defaut).** L'ordre des films tourne d'une passe a
   l'autre : chaque film occupe toutes les positions de la boucle. Un seul film est en memoire a la
   fois ; la memoire est rendue au systeme entre deux films, hors chronometre. Un ramasse-miettes force
   precede chaque phase mesuree.
4. **Empreintes hors chronometre**, calculees par le fils apres la lecture du pic.
5. **Mediane et etendue** par phase ; la mediane de passes en nombre pair est la moyenne des deux
   valeurs centrales.
6. **Regle de dispersion.** Si deux releves consecutifs d'un meme code different de plus de 5 % sur le
   total median d'un film, refaire le releve avec 9 passes, puis s'arreter : la dispersion se range
   telle quelle.
7. **Fenetre de mesure.** Releve pris machine au repos : aucun autre travail lourd, aucun test ni
   compilation en cours. Le journal du decodeur est reduit aux erreurs pendant la mesure.

## Releves par version

Le releve d'une version se range dans `bench/releves/vX.Y.Z.tsv`, commite avec la version.

| Version | Total median (ms) par film | Pic (Mio) |
|---|---|---|
| a remplir avec le releve de reference | | |

## La suite des etapes (`-etapes`)

`-etapes <fichier>` prouve que le parallelisme ne change ni le contenu ni l'ordre de ce que le decodeur
publie. Apres les mesures, l'outil refait UNE passe par film, hors de tout chronometre (ses durees sont
jetees, aucune colonne du releve n'en vient), avec le meme `-fils`. Le fichier porte une ligne par etape
observee (`Options.Observe`), dans l'ordre ou le decodeur la publie : `film`, `rang`, `etape`, `empreinte`
(SHA-256 hexadecimal de la valeur de l'etape serialisee en JSON). Une valeur que JSON ne sait pas ecrire
porte une empreinte prefixee `erreur:` : elle reste comparable d'un releve a l'autre mais ne prouve que
l'echec de serialisation. Deux fichiers produits a des nombres de fils differents doivent etre
identiques, a l'en-tete pres (la ligne `# fils`).
