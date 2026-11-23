package replay

// filmfacts_schemas.go — LA VERSION DE LA CHARGE DES FAITS ET L HISTOIRE DE SES MONTEES, sortie de `filmfacts_fichier.go` (plafond de 500 lignes).

// SchemaDesFaits est la version de la CHARGE : quelles sections existent et ce qu elles portent.
// Elle monte quand une section nait, meurt ou change de contenu.
//
// DISTINCTE DE [VersionCodecFaits] parce que les deux ne commandent pas la meme decision : un
// lecteur qui ne connait pas le CONTENEUR ne sait rien lire ; un lecteur qui connait le conteneur
// mais pas le SCHEMA sait lire l en-tete, donc sait dire « perime » proprement.
// SCHEMA 2 (2026-09-18, lot 4.1.3) : la CHARGE de la section 1 a change de forme — le blob des
// entrees passe en v23 (pistes d objets du monde en float32 exact, record de creation entier,
// denominateurs entiers). C EST CE NUMERO QUI DOIT PORTER LE CHANGEMENT, et pas la seule magie du
// blob : la magie vit DANS la section 1, donc un fichier d une version anterieure passerait le
// verdict de fraicheur (l en-tete est identique, les quatre revisions n ont pas bouge) et ne
// serait refuse qu au decodage de la section, par le chemin « illisible malgre un en-tete frais ».
// Un fichier PERIME doit se dire perime SUR SON EN-TETE, en 110 octets, pas apres avoir ete lu
// jusqu au mega-octet de positions. [TestFaitsDUnSchemaAnterieurSontRefusesSurLEnTete] le prouve.
// SCHEMA 3 (2026-09-19, post-chantier lot 5.1) : la CHARGE de la section 1 change encore — le blob
// des entrees passe en v24 (la JAUGE DE RETOUR du drapeau et son temoin). Meme raisonnement qu au
// schema 2 : le refus doit tomber sur l EN-TETE, pas au decodage de la section.
// SCHEMA 4 (2026-09-24, integration de la vague D de la campagne « retours rejeu » : UNE montee
// pour les lots M2 et M3, qui avaient chacun pose 4 sur leur branche). Lot M3 : la CHARGE de la
// section 1 change — le blob passe en v26 (la sante de la marche d image-cle, puis les dotations
// de naissance lues dans le record NEW du bipede). Lot M2.2 : le complement de la section 1 porte
// LES OCCUPANTS DU MATCH (`FilmInputs.PlayerEntities`, une entree par entite `ti=9`), et la
// section 5 les INSTANTS de BOT_METADATA (`BotEntry.Declarations`). Meme raisonnement qu aux
// schemas 2 et 3 : le refus tombe sur l EN-TETE. Un fichier du schema 3 n a ni les uns ni les
// autres ; la grammaire monte avec (`grammar.Rev`) : les faits d avant sont PERIMES, il faut
// redecoder. LOT M8 (2026-09-24), MEME SCHEMA 4 (montee de la vague D, pas encore publiee) : le
// complement de la section 1 porte en dernier le VERDICT DU FIL DES MORTS et sa cause
// (`FilmInputs.DeathsFeed`). Un fichier au schema 4 ecrit avant ce lot bute sur ce verdict
// absent : relu « illisible malgre un en-tete frais », il est redecode, jamais servi sans verdict.
// LE MEME SCHEMA 4 PORTE LE LOT M4b (2026-09-24, meme vague D, jamais publiee) : le blob passe en
// v27 (tirs lus par la grammaire du record — indice sur cinq bits, numero de tir, unite tireuse — et
// le TIR CONTINU de la vue de controle). Un fichier ecrit par le code de la vague D d avant M4b porte
// le blob v26 : il est refuse a la magie du blob, et redecode.
// LE MEME SCHEMA 4 AU JALON J3 (2026-09-26) : le complement de la section 1 gagne, avant le verdict
// du fil des morts, le TEMOIN DE L INVENTAIRE (nul = illisible, lot J3.6) SOUS LA MONTEE DU CODEC 2 (plan, J3.6 :
// « meme montee que J3.4 ») — le codec 2 n a jamais ete PUBLIE sans lui (les etats intermediaires du lot n ont ecrit aucun fait dans un dossier de donnees), et un fichier du codec 1
// est refuse sur son prefixe.
// LE MEME SCHEMA 4 AU JALON J11.0 (2026-09-28) : le blob passe en v28 (cinq compteurs de
// `MovementStateStats` que le blob ne portait pas, dont les deux publies du saut) SOUS LA MEME
// MONTEE DU CODEC 2, toujours non publiee. Un fichier ecrit par un binaire de la branche avant
// J11.0 porte le blob v27 : il est refuse a la magie du blob, et redecode.
// SCHEMA 5 (2026-10-06, lot VA de la campagne de grammaire, decision du pilote du 2026-10-06) : la
// section 2, l identite du film (JSON de `profile.FilmIdentity`), change de contenu. L etape V1 y a
// ajoute la simulation de l enregistreur (`SimulationDeLEnregistreur`, `OptionsDePartieLues`), l etape
// V3 la variante de partie (`Variante`). Relue par `json.Unmarshal`, une section 2 du schema 4 rendrait
// ces champs a zero sans erreur : c est le refus SUR L EN-TETE qui doit le dire, comme aux schemas 2 a
// 4. La montee de `grammar.Rev` et de `profile.Rev` du meme lot refusait deja tout fichier anterieur
// sur ses revisions de couche, donc aucun fait perime n etait servi ; le schema tient sa propre
// doctrine (« il monte quand une section change de contenu ») au lieu de s en remettre a elles. Le
// lot 5.18.1 (`ControleDeCorruption`, 2026-09-22) avait elargi la meme section sans le monter : tout
// fichier du schema 4 est desormais refuse sur son en-tete, celui-la compris.
// SCHEMA 6 (2026-10-06, lot « Rejeu : toute entree du roster a l equipe que le film ecrit ») : la
// section 5 porte L EQUIPE DE CHAQUE BOT lue dans BOT_METADATA (`BotEntry.Team`) et le bilan de sa
// lecture (`Roster.BotEquipes`) ; la liaison des occupants en fait l equipe des bots qu aucune
// entite `ti=9` ne porte. Les faits du schema 5 (vue A) ne la portent pas. `killsource.Rev` ne monte
// pas (aucune ligne de kill ne change, cf. sa chronique) : c est ce numero qui refuse, SUR L EN-TETE,
// un fichier d avant — sans lui, des faits « frais » sans equipe de bot se rejoueraient, et le
// correctif n atteindrait aucun artefact.
// SCHEMA 7 (2026-10-07, branche `feat/zones-etat-initial`) : la section 1 porte, a la suite des
// lectures delta de l etat des zones, ses lectures d IMAGE-CLE (`FilmInputs.ZoneKeyReads`,
// `encodeGardesDeMode`). Le blob ne change pas ; ce que la section porte, si.
// SCHEMA 9 (2026-10-07, branche `feat/zones-proprietaire`) : chaque lecture de `ti=13` de la section 1
// porte le NOM de sa propriete (`grammar.ManagedPropertyRead.Name` / `Named`, renseigne aux
// images-cles) ; un fichier d avant se relirait decale. Le rang 8 est reserve par un lot parallele
// non fusionne.
// SCHEMA 10 (2026-10-07, lot 2.7.c de la representation intermediaire) : la section 5, le resultat de
// killsource, change de contenu : ses comptes de replis du decodage (`ReplisDuDecodage`) perdent
// `HorsBandeBipede` (repli retire avec la bande bipede, 2.7.c2) et gagnent `KillsRattrapes`
// (rattrapage des kills hors de la vue A lue, 2.7.c4). Le rang 8, reserve a ce lot, reste sans
// emploi : la serie suit les fusions.
// SCHEMA 11 (2026-10-09) : la
// section 5 change de contenu. Le profil calibre qu elle porte (`killsource.Result.ProfilCalibre`,
// `grammar.GrammaireBalayage`) perd les bascules `PorteeBaseline` et `GrammaireEcrivainI0`, retirees
// quand la portee de l etat complet est devenue un champ du lecteur, et gagne `MoteurUnPossible`
// (le type de moteur que le film n exclut pas). Mesure : `replay-equiv` sur les 20 films, binaires de
// `d429dc517` contre ceux du lot, etape `killsource` differente sur les 20 films, document publie
// identique hors les valeurs de revision. La montee de `grammar.Rev` refusait deja tout fichier
// anterieur sur ses revisions de couche ; le schema tient sa doctrine.
// SCHEMA 12 (2026-10-14, lot F du plan d ajustements) : le complement de la section 1 porte, avant le
// verdict du fil des morts, les POUSSEES d equipement (`FilmInputs.Knockbacks`, `KnockbackStats`).
// SCHEMA 13 (2026-10-14, lot « equipement porte » du plan d ajustements) : le complement de la section 1
// porte, apres les poussees, les lectures de l EQUIPEMENT PORTE (`FilmInputs.CarriedEquipment`,
// `CarriedEquipmentStats`).
// SCHEMA 14 (2026-10-15, meme lot, fin du champ de reparation) : chaque lecture de l equipement porte
// porte les vies que sa pile i28 designe (`Tracked`), les suppressions de ces vies se lisent
// (`tracked-end`), et deux denominateurs s ajoutent (`TrackedEntries`, `TrackedEnds`).
// SCHEMA 15 (2026-10-15) : la section 5 change de
// contenu. Le profil calibre qu elle porte (`grammar.GrammaireBalayage`) gagne
// `MalleableSansDrapeauFinal` (le bloc malleable d i53 sans drapeau final des films de version majeure
// declaree 0x25). Mesure : `replay-equiv` sur les 20 films, binaires de `feat/v75` contre ceux de la
// branche, etape `killsource` differente sur les 20 films.
// SCHEMA 16 (2026-10-16, integration groupee `feat/integ-film` de deux lots qui avaient chacun pris 16
// sur leur branche ; UNE montee) : (1) lot K-B (`fix/ks-b-equipe-bots`) — la section 5 change de
// contenu, le bilan de la lecture de l equipe des bots (`killsource.EquipesDesBots`) gagne
// `FragmentsEcartes` et `PositionInconnue` ; (2) lot « faille a la pose » — chaque lecture de
// l equipement porte porte la creation lue des objets que sa pile nomme pour la premiere fois
// (`Spawned` : instant, archetype, position i0), et deux denominateurs s ajoutent (`SpawnedBirths`,
// `SpawnedPositioned`). Un fichier cuit sur l une des deux branches porte d autres revisions de couche
// (`grammar-2026-10-15.3`) et se refuse sur elles.
// SCHEMA 17 (2026-10-19, correctif des vehicules d arene) : la section 1 porte `VehicleScan.Scanned`, et
// il change de SENS pour un film dont les images-cles sont lues sans slot `ti=40` (une arene) : vrai
// (zero vehicule mesure) au lieu de faux (« pas lu »). Aucune revision de couche ne bouge (aucun bit n est
// lu autrement) : c est ce numero qui refuse, SUR L EN-TETE, les faits d avant, sans quoi la recuisson les
// rejouerait et le correctif n atteindrait aucun artefact (meme raisonnement qu au schema 6).
// SCHEMA 18 (2026-10-22, lot detonations) : le blob des entrees passe en v29 — les messages de
// detonation/impact de projectile lus dans la vue A (`FilmInputs.Detonations`, genres 5/6,
// `encodeDetonations`) entrent a la queue de la section des projectiles. SOUS LA MEME MONTEE DU CODEC 2
// (le cadre du conteneur ne change pas, seule la charge de la section 1 grandit). Un fichier ecrit avant
// ce lot porte le blob v28 : il est refuse a la magie du blob, et redecode — sans quoi un artefact rejoue
// depuis des faits perimes perdrait la detonation lue et retomberait systematiquement sur le repli nomme
// (`fallback.NomDetonationDernierePositionConnue`).
// SCHEMA 19 (2026-11-08, equipement porte) : chaque lecture de l equipement porte porte l IDENTITE de l objet a sa
// creation (`CarriedEquipmentRead.HasGlobalID`, `GlobalID` : le GlobalID de son tag `eqip`), ecrite apres le genre de
// lecture. La famille de l objet se lit desormais sur lui, et non sur l annonce i48 du porteur dans le meme paquet.
// `grammar.Rev` monte avec ce numero (`grammar-2026-11-08`) : un fichier ecrit avant ce lot est refuse sur les deux.
// SCHEMA 20 (2026-11-12, infos de match) : la section 2, l identite du film (JSON de `profile.FilmIdentity`),
// porte `InfosDeMatch` : la variante de partie (nom, description, score de victoire, duree, delai de
// reapparition, prolongation, dotation, options), les identifiants de la variante de carte et l heure de debut.
// Relue par `json.Unmarshal`, une section 2 du schema 19 rendrait ces champs absents sans erreur : le refus SUR L EN-TETE le dit. Aucun consommateur : `SchemaVersion` ne bouge pas.
// SCHEMA 21 (2026-11-12, occupants tues lus) : la section 1 porte les dead-states des BIPEDES (`VehicleScan.BipedDeaths`, `ti=35` : victime,
// tueur, effet de degat), a la suite de ceux des vehicules ; ils disent QUI est mort avec un vehicule detruit. Une charge du schema 20 les
// rendrait vides sans erreur (« personne n est mort avec son vehicule ») : le refus SUR L EN-TETE le dit. Aucune revision de couche ne bouge.
// SCHEMA 22 (2026-11-12, creation des records NEW et charges de la vue A selon le build) : la section 5, le resultat de
// killsource, change de contenu. Le profil calibre qu elle porte (`killsource.Result.ProfilCalibre`,
// `grammar.GrammaireBalayage`) gagne `CreationParPorte` et `ProprieteGereeComplete` : ce que l ecrivain du build du film ecrit d un
// record NEW (`profile.EcritureDuBuild`, `profile-2026-11-12.2`). Relu par `json.Unmarshal`, un profil du schema 21 rendrait ces deux
// bascules fausses sans erreur, et un profil qui les croit fausses lit un film HI_1_11_0 a HI_1_13_0 comme avant : c est le refus SUR
// L EN-TETE qui doit le dire. `grammar.Rev` (`grammar-2026-11-12.4`) et `profile.Rev` montent avec ce numero, comme aux schemas 11 et 15.
// SCHEMAS 24 ET 25 : la section 1 porte les plages de listage (type 8), les lectures d `unit-control`, les horloges et les comptes a rebours ; la section 2 les regles de la variante ; `grammar.Rev` et `profile.Rev` montent.
// SCHEMA 26 : le complement de la section 1 porte aussi les messages de jeu et les noms de script du chunk 0 (`FilmInputs.GameEvents`, `Scripts`) ; `grammar.Rev` monte.
// SCHEMA 27 : le complement de la section 1 porte aussi les entrees des entites du tableau des scores (de qui est chaque ligne) ; `grammar.Rev` monte.
// SCHEMA 28 : le complement de la section 1 porte aussi les suppressions de bipede (`FilmInputs.BipedDeletions`, les records DEL des bipedes lus par la marche des trames) et leurs comptes ; `grammar.Rev` monte.
// SCHEMA 29 : le complement de la section 1 porte aussi la jauge de retour lue par la marche (`FilmInputs.FlagGaugeWalk`), les messages `damage_aftermath` (`FilmInputs.Damages`) leurs points (`DamagePoints`) et le compte des trames de vitalites non lues ; `grammar.Rev` monte.
// SCHEMA 30 : le complement de la section 1 porte aussi les rechargements d arme (`FilmInputs.Reloads`, genre 38 de la vue A) et les commandes de deplacement tenues (`FilmInputs.ControlHolds`, vue C), deux canaux de la marche qui suivent les degats ; chaque creation d equipement porte son parent (`EquipmentCreation.HasParent`, `Parent`), le chassis qu une piece montee annonce a sa creation. `grammar.Rev` monte.
// SCHEMA 31 : le complement de la section 1 porte aussi les repliques des Spartans et les annonces (`FilmInputs.Repliques`, genres 75 et 76 de la vue A), les gestes du joueur (`FilmInputs.Gestures`, genres 39 et 40 de la vue A : lancer amorce et coup de melee) et, pour chaque rechargement, le nombre de copies de son message retirees par le canal (`ReloadRead.Copies`) ; les commandes tenues portent leur genre (`ControlHold.Kind` : deplacement, accroupi, saut) et leurs comptes par genre (`ControlHoldStats.Move`, `Crouch`, `Jump`) ; le complement porte le pas des entrees de controle (`Gaps`, `StepGaps`, `StepUS`). `grammar.Rev` et `objectives.Rev` montent. Le meme numero porte enfin les munitions de chaque delta d inventaire (`InventoryDelta.Ammo` : chargeur, fraction R(12) brute, reserve), ecrites a la fin du delta et relues ; l assemblage les lit (`ammoReads[]`, `ammo_reads.go`), et la magie du blob d entrees passe a `REPLAYINPUTS31`. Le complement porte enfin le jeu d armes desire du bipede (`FilmInputs.DrawnWeapons`, `DrawnWeaponStats` : drapeaux `Scanned` et `Absent`, comptes lus, slot non lie et doublons, puis chaque lecture : horodatage en delta, slot, emplacement principal et seconde main), apres les commandes tenues (`filmfacts_arme_tiree.go`) ; et chaque lancer de grenade porte le handle du projectile qu il cree (`GrenadeThrow.HasLife`, `Life.Slot`, `Life.Gen`) : section des lancers `ts, film_index, type_id, bool8 has_life, [slot, gen]`. Le complement porte enfin les changements de sante et de bouclier par vie (`FilmInputs.Vitals`, `VitalStats`), apres l arme tiree (`filmfacts_vitalites.go`). Le CONTENU de la section des visees des vehicules change : un film sans vehicule y porte ses visees sans position (500 a 9 000 lectures), lues sur tout film ; le codec ne change pas.
const SchemaDesFaits = 31
