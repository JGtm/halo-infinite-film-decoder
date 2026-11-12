package grammar

// rev_chronique.go — LA CHRONIQUE DE [Rev], UNE ENTREE PAR RANG.
//
// # POURQUOI CE FICHIER EXISTE (2026-09-18, lot 2.4.1)
//
// La chronique vivait dans le godoc de [Rev]. Au rang `.28` ce fichier passait 500
// lignes, et le ratchet de taille (`archlint/film_file_size_test.go`) le refusait — a juste
// titre : une chronique qui ne peut plus grandir cesse d etre tenue, et c est exactement le
// defaut F5 que `TestChroniqueCouvreLaRevisionCourante` a ete ecrit pour fermer. Elle vit donc
// ici, ou elle peut grandir, et `rev.go` ne garde que la regle et la constante.
//
// # CE FICHIER N EST PAS DE LA GRAMMAIRE
//
// Comme `rev.go`, il est EXCLU de l ensemble hache par l empreinte (cf.
// `fichiersHorsGrammaire`) : il DECRIT la grammaire, il n en fait pas partie. Sans cette
// exclusion, ecrire une entree changerait l empreinte, et la branche « la revision a change
// sans que la grammaire bouge » redeviendrait du code mort (revue R1, P2-3).
//
// # LA CHRONIQUE, A PARTIR DU `.11` : UNE ENTREE PAR RANG, ET RIEN QU UNE
//
// LES TROIS DERNIERES ENTREES ONT ETE REECRITES LE 2026-09-16 (revue de jalon M1, RONDE 2,
// constat F5). Elles etaient EMPILEES et toutes trois annoncaient « `.11` -> `.12` », suivies de
// deux lignes « FUSION ... au rang suivant » qui racontaient une renumerotation que l integration
// n a jamais faite : la chronique s arretait donc a `.12` pendant que la constante valait `.14`,
// et les changements de COMPORTEMENT portes par `.13` et `.14` n avaient AUCUNE entree. Relevee
// commit par commit sur l integration (`git log --first-parent`), la suite reelle est celle-ci —
// un lot, un rang, dans l ordre ou les merges sont tombes.
//
// LES RANGS ANCIENS VIVENT DANS LES ARCHIVES : `.12` a `.28` dans `rev_chronique_archive.go`,
// `.29` a `.42` dans `rev_chronique_archive_2.go`, `grammar-2026-09-18.2` et `.3` dans
// `rev_chronique_archive_3.go`, `grammar-2026-09-20` a `.2` et `grammar-2026-09-21` a `.4` dans
// `rev_chronique_archive_4.go`, `grammar-2026-09-21.5` a `grammar-2026-09-22.6` dans
// `rev_chronique_archive_5.go`, `grammar-2026-09-22.7` a `.12` dans `rev_chronique_archive_6.go`,
// `grammar-2026-09-24` a `grammar-2026-10-02.3` dans `rev_chronique_archive_7.go`,
// `grammar-2026-10-03` a `grammar-2026-10-08` dans `rev_chronique_archive_8.go`, `grammar-2026-10-08.2` a
// `grammar-2026-10-28` dans `rev_chronique_archive_9.go`.
// La chronique se ROTATIONNE quand ce fichier
// atteint 500 lignes ; le geste a ete refait le 2026-09-16 (lot
// 2.5.b, rangs `.21` a `.26`), le 2026-09-18 (lot 5.1.7, rangs `.29` a `.38`, dans une
// SECONDE archive parce que la premiere frolait le meme seuil), le 2026-09-21 (lot 5.9.4,
// rangs `.39` a `.42`, verses dans cette meme seconde archive qui avait la place), puis le
// 2026-09-22 (lot 5.20.1, une CINQUIEME archive), puis le 2026-09-24 (lot M4b, une SIXIEME), puis
// le 2026-10-03 (integration de la vague 1 de la campagne de grammaire, une SEPTIEME), puis le
// 2026-10-06 (lot VA de la campagne, rangs `grammar-2026-09-27` a `.3` verses dans la septieme), puis le
// 2026-10-07 (fusion de `feat/v75` dans le lot des arrets de la vue B, rangs `grammar-2026-10-02` a
// `.3` verses dans la septieme), puis le 2026-10-08 (lot des arrets de la vue B, suite, une
// HUITIEME archive), puis le 2026-11-12 (plan de la marche complete, A2.H, une NEUVIEME).
// C est le
// geste ordinaire que l en-tete des archives annonce, pas un incident. Ce qui suit est la suite
// VIVANTE, a partir du `grammar-2026-11-08`.

// ENTREE `grammar-2026-11-08` (2026-11-08, equipement porte : l identite se lit sur l objet). LE CANAL DE
// L EQUIPEMENT PORTE REND, A LA CREATION DE CHAQUE OBJET ATTACHE, SON IDENTITE : le GlobalID de son tag `eqip`,
// le mot de 32 bits inconditionnel du bloc `object-multiplayer-properties` de son etat par defaut
// ([MPPWord32], relu par [lecteurDIdentite] avec le deserialiseur de production [consumeDefaultStateTI37], a
// `HeaderBit` + [woNewHeaderBits] du record NEW). Aucun bit n est lu autrement : le mot etait lu et jete par la
// marche. Mesure sur 1 448 films : 293 693 objets attaches, 0 sans identifiant, 0 echec de lecture ; 46 505 objets
// actives, 46 478 resolus par le catalogue de l energie des equipements et le manifeste. La creation
// (`CarriedEquipmentRead`) gagne `GlobalID` et `HasGlobalID`, et le rejeu en tire la famille de l objet quelle que
// soit la facon dont le joueur l a obtenu (ramasse, equipement de reapparition, equipement de depart de match) :
// la jointure par le rang de l annonce i48 du meme paquet, qui perdait les deux derniers cas, disparait. Les
// lectures changent de forme (le codec des faits les ecrit avec l identite, `SchemaDesFaits` 18 -> 19) : les faits
// cuits avant se refusent sur cette revision et sur ce numero, par construction.

// ENTREE `grammar-2026-11-10` (2026-11-10, lecture des positions par index de region). LES POSITIONS SONT LUES
// AUX LARGEURS DE LA REGION QUE LE FLUX DESIGNE. LE DECODAGE CHANGE. `FUN_14076e524` lit un index de region
// puis trois axes aux largeurs de la ligne de CET index ; le portage unique ([lireE524Sur]) appliquait a tout
// index la ligne de la region jouee. Il lit desormais la table par index de la carte
// ([profile.MapQuantEntry.Regions], posee par [ProfilDeBalayage.PoserLesRegionsDeLaCarte]) : un index sans
// ligne lit zero bit par axe et se compte ([Observation.IndexSansLigne]) ; une entree sans table garde la lecture
// d avant. Trois sites rejoignent le portage unique et quittent les exceptions datees : flock-position (ti=21
// i16), flock-destination (ti=21 i2 a i11) et world-object i0, branche precHigh = 0 ; seule la branche precHigh = 1
// de world-object i0 reste en exception. Les positions des messages de la vue A (genres 5 et 6, niveaux 0xf et
// 0xc) lisent la meme table, et la position d une detonation se dequantifie dans la region lue. Mesure sur le
// corpus (127 films lus, catalogue regenere) : paquets delta fermes HI_1_13_0 97,60 % -> 99,42 %, HI_1_11_0
// 96,07 % -> 98,29 %, HI_1_10_0 95,41 % -> 95,46 %, HI_1_12_0 inchange ; le desaccord de lecture contre le temoin
// `ti=40` composant 0 disparait (12 087 paquets). Les rejeux et les faits cuits avant se refusent sur cette
// revision : la recuisson est une decision de l utilisateur.

// ENTREE `grammar-2026-11-10.2` (2026-11-10, type d une entite lie a son eid complet). LE TYPE D UNE ENTITE EST
// LIE A SON EID COMPLET, SLOT ET GENERATION. LE DECODAGE CHANGE. La garde de delta du jeu (`FUN_1406cbaa0` cas 3,
// `FUN_1406caad8`) compare `*(uint *)(slot * 200 + t)` a l eid entier et ne lit aucun bit de corps quand ils
// different ; la boucle de records de la vue B ne testait que « slot lie » ([rejetDeVue]). Elle consulte
// desormais [World.ArchetypeForEID] : une liaison a generation connue (NEW lu, inference) ne repond qu a son eid,
// une liaison de generation inconnue (image-cle, table de datums) repond a toute generation de son slot, et le repli
// d anticipation ([World.LierParRepliDAnticipation]) traite un slot lie sous une autre generation comme un eid non
// lie. Aucun type n est devine : un delta d une generation dont la creation n a pas ete lue sort de la vue B par
// rejet hors datum, comme chez le jeu. Mesure sur 38 films (30 recents, 8 anciens) : aucun compte de paquets fermes
// ne bouge, la garde se declenche 1 090 fois (marches d essai comprises) ; sur la mini-bobine de 000d5950 des
// records lus sous un type perime ne se lisent plus. Les rejeux et les faits cuits avant se refusent sur cette
// revision : la recuisson est une decision de l utilisateur.

// ENTREE `grammar-2026-11-10.3` (2026-11-10, `biped-spartan-ability`, tag 3). LA PORTE DU CORPS PORTE SUR LES 6 BITS
// LUS DANS LE FLUX. LE DECODAGE CHANGE. `FUN_142f262d4` met `dst[2]` a zero en tete, puis `FUN_14297ea84` y range les 6
// bits qu elle lit : le test `dst[2] & 1` porte sur ces bits, et la branche qu il garde est atteignable. Le lecteur
// ([consumeSpartanAbilityTag3]) lit la suite du jeu : a ; si a, flags = R(6) ; si flags & 1, c = R(1), puis la position
// de `FUN_14076e494` (c nul) ou la reference de categorie 0 de `FUN_1406d3140` (13 bits de valeur et 2 de queue, soit 15
// bits) suivie, sous flags & 0x10, du corps de `FUN_142f04664` (c = 1) ; puis b = R(1) et, si b, la seconde position.
// Mesure sur 38 films (30 recents, 8 anciens) : aucun film ne ferme moins de paquets ; HI_1_13_0 98,79 % -> 98,90 %,
// HI_1_12_0 97,70 % -> 97,75 %, et les quatre builds anciens gagnent de 30 a 159 paquets. Les rejeux et les faits cuits
// avant se refusent sur cette revision : la recuisson est une decision de l utilisateur.

// ENTREE `grammar-2026-11-12` (2026-11-12, la classe de la vue A vient de la seule table des genres ; trois charges
// de la vue A). LE DECODAGE CHANGE. La classe d un film (EGALE, PREFIXE, ILLISIBLE) se decidait sous la version
// majeure que l executable lu joue (0x29, `FUN_1428e219c`) : un film de table egale sous une autre majeure, ceux
// de HI_1_12_0 (0x28), etait classe PREFIXE, si bien que la fin de sa vue A n etait acceptee que si la marche
// depuis ce bit fermait le paquet ; sinon la marche retombait sur une localisation fausse, perdait les creations
// du paquet, et des rejets hors datum suivaient en cascade. La garde decidait de ce que le jeu joue, pas de ce que
// l ecrivain du film a ecrit ; la decision du 2026-10-06 la posait sur ce principe, sans mesure d un effet sur la
// fermeture (elle avait fait passer le gain de bcb6d393 de +1 239 a +599). La classe ne depend plus que de la table
// declaree ([classeDesGenres]) : [classeSousLaMajeure] et [versionMajeureJouee] sont supprimees. La table des huit
// films HI_1_12_0 de la mesure est identique a la native, genre par genre (123 genres) ; aucun film d une autre
// majeure ne change de classe (les tables des majeures 0x25 a 0x27 et du HI_1_11_0, 122 genres, ne sont pas egales
// a la native). Trois charges de la vue A, de disposition MESUREE contre le temoin et non relue dans le jeu, sont
// portees ([chargeChocDeMelee] genre 12, 112 bits ; [chargeEffetSurObjet] genre 14, 53 bits ;
// [chargeCollisionDObjet] genre 106, 82 bits, `vue_a_charges_mesurees.go`) : les douze paquets de huit films
// HI_1_12_0 que la vue A arretait sur l un d eux se lisent jusqu a leur terminateur. Mesure sous carte, 38 films
// (30 recents, 8 anciens) : paquets fermes HI_1_12_0 220 599 -> 225 278 (97,75 % -> 99,83 %), dont 4 672 pour la
// classe et 7 pour les charges ; HI_1_13_0 822 612 -> 822 680, HI_1_11_0 +5, HI_1_10_0 +3, HI_1_9_0 +1, HI_1_8_0
// inchange ; fermetures fausses (fermes chez nous, position differente du temoin) 57 -> 1 sur les huit films
// HI_1_12_0 et 37 -> 4 sur les dix temoins. Trois films baissent : 6ceaf3b8 de 6, 972cdbc6 et aa00a8cd de 1, des
// paquets que leur fermeture fausse faisait compter fermes ; ils s arretent desormais sur l etat de creation d un
// archetype que le decodeur ne porte pas (ti=13, 47, 10). Les rejeux et les faits cuits avant se refusent sur cette
// revision : la recuisson est une decision de l utilisateur.

// ENTREE `grammar-2026-11-12.2` (2026-11-12, la table d entites d une image-cle se retrouve par les invariants de ses en-tetes). LE
// DECODAGE CHANGE. Le jeu lit la table d entites du morceau par `FUN_142e2bfd0` : une entree par emplacement, dans l ordre des
// emplacements, vides compris, dont l en-tete de 108 bits (eid, type, poignee, drapeaux, octet) donne l eid COMPLET et le type que la
// garde de delta de `FUN_1406cbaa0` compare. Le decodeur la retrouvait en cherchant, a chaque bit, un mot `[eid][type]` aux gardes de
// la marche d ancres (generation non nulle, type sous 50) et en gardant la plus longue suite croissante d emplacements : les entites de
// generation 0, ecartees par la garde, manquaient (309 sur 337), un mot pris dans les donnees d une autre entite donnait un faux type
// (13) et 276 liaisons n avaient pas d entree. La recherche est desormais celle des invariants de l en-tete entier ([EntetesDeTable]) :
// emplacement sous 8192, type sous 64, octet sans ses trois bits de poids fort, drapeaux 0 (aucune poignee, n et m nuls) ou 1 et 3 (n
// au plus 256), emplacements strictement croissants ; une premiere passe en generation non nulle, une seconde pour la generation 0 entre
// deux entrees reconnues. Les donnees de creation et d etat des entrees ne se lisent pas. La liaison que la table pose porte
// desormais son eid complet ([World.BindEntreeDeTable]) : elle ne repond qu a cet eid ([World.ArchetypeForEID]), et remplace une
// liaison de la chaine d ancres ou d un NEW qui dit autre chose ; les liaisons de generation inconnue (chaine d ancres, joker,
// anticipation) sont inchangees. La borne de n remplace la taille de creation par type du decodeur tiers, propre a HI_1_12_0 et
// HI_1_13_0 (quatre octets de moins pour sept types jusqu a HI_1_11_0) : sans elle la recherche perdait des entrees sur les builds
// anciens. Mesure contre la table du decodeur tiers (5 films, 102 157 entrees) : 102 155 justes, 2 manquees, 0 fausse ; entrees
// absentes a l ouverture d un morceau 313 -> 0, au faux type 13 -> 0, paquets de mise a jour visant une entite absente 5 743 -> 0 et
// une entite au faux type 1 310 -> 0 (217 431 paquets). Mesure sous carte, 38 films : paquets fermes HI_1_13_0 98,91 % -> 99,85 %,
// HI_1_12_0 99,83 % -> 99,95 %, HI_1_11_0 98,37 % -> 99,85 %, HI_1_10_0 95,72 % -> 96,56 %, HI_1_9_0 92,66 % -> 98,81 %, HI_1_8_0
// 69,71 % -> 73,43 % ; aucun film ne baisse ; fermetures fausses des dix temoins 4 -> 3. La generation connue ne change aucun
// verdict de paquet sur ces films (meme fermeture que la liaison de generation inconnue). `DatumAmbiguous` change de sens : il compte les
// emplacements dont un en-tete valide a ete ecarte par la croissance des emplacements (6 000 a 9 600 par film avec la recherche d avant,
// 200 a 300 avec celle-ci). Les rejeux et les faits cuits avant se refusent sur cette revision : la recuisson est une decision de
// l utilisateur.

// ENTREE `grammar-2026-11-12.3` (2026-11-12, infos de match lues dans `chunk_00`). LA LECTURE DE L IDENTITE RENDU DE NOUVEAUX CHAMPS ;
// AUCUN BIT DEJA LU NE CHANGE D INTERPRETATION. Le corps de `chunk_00` porte la variante de partie en Bond CompactBinary : la marche
// du corps ne lisait que trois de ses champs ([lireLaVarianteDePartie]). Une lecture par chemins ([lecteurDuCorps.valeurBond], un
// message Bond parcouru de bout en bout, les valeurs hors chemin enjambees par leur longueur) rend desormais le nom et la description
// (la premiere langue de la table de chaines de la variante, trouvee par son decalage), le score de victoire, la duree, le delai de
// reapparition, la prolongation et la derniere chance, la dotation du premier chargement et les options ([lireLesInfosDeMatch]) ; les
// deux blocs de 16 octets qui ferment la section d identification sont l identifiant d asset et l identifiant de version de la
// variante de carte, et l heure de debut est celle d avant. Un champ que le film n ecrit pas reste absent. Disposition des chemins
// mesuree contre le decodeur tiers, sans cas particulier par build. Mesure sur les 2 123 films du cache : 2 118 identites lues, la
// variante lue sur les 2 118 (HI_1_4_1 a HI_1_13_0), les cinq autres sans section d identification ; accord avec le decodeur tiers sur
// chaque champ des 1 791 films qu il lit (un seul ecart : le nom et la description d un film dont la table de chaines n ecrit qu une
// langue) ; identifiant de carte egal a celui du registre des matchs sur 1 457 films sur 1 457. La section 2 des faits porte ces champs
// (`replay.SchemaDesFaits` 20) : les faits cuits avant se refusent sur ce numero et sur cette revision, la recuisson est une decision
// de l utilisateur. Le document de rejeu ne change pas.
