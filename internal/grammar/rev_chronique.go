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

// ENTREE `grammar-2026-11-12.4` (2026-11-12, la creation d un record NEW et les charges de la vue A se lisent selon le build ; la
// marche de la vue B ne s arrete plus sur une porte a 0). LE DECODAGE CHANGE. Un record NEW lisait toujours la porte de composants
// puis le masque ([lireLaPorteEtLeMasque]) ; `FUN_1408f1aa4` saute la boucle de composants, masque compris, sous une porte a 0, et
// ne lit pas de porte pour un archetype dont le masque de creation est vide (`vtable[0x30]`, `FUN_1408f2130` : ti=7, 13, 14, 20, 23,
// 28, 29, 46, 47, 49). Une creation de ti=10 sans etat initial (42 bits chez le decodeur tiers) se lisait sur 122, la marche de la
// vue B s arretait trois records plus loin, et la vue C etait lue au mauvais endroit : 35 paquets que v1.0.1 lisait juste ne l etaient
// plus sur cinq films HI_1_12_0 (11, 1, 8, 14 et 1), la fin de leur vue A etant pourtant le debut de leur vue B. L etat de creation de
// ti=13 lit la charge de ses etiquettes ([consumeDefaultStateTI13Creation]) ; les trois charges de la vue A de disposition mesuree
// (genres 12, 14 et 106) ne se lisent que sur un film dont le build les etablit ([chargeDuGenre], [grammaireDeLaVueA.chargesMesurees]) :
// appliquees a HI_1_10_0 et HI_1_9_0 elles lisaient les messages suivants (85, 82, 5, 6, 104) a un mauvais alignement, dont huit
// poussees fausses sur `084a804d` (reference 0 absente). Les trois dispositions viennent du profil du build ([profile.EcritureDuBuild],
// `profile-2026-11-12.2`) : HI_1_11_0, HI_1_12_0 et HI_1_13_0 ; un autre build garde la lecture d avant la porte et la vue A s arrete sur
// le message d un des trois genres, comme avant `grammar-2026-11-12`. Mesure sous carte, 223 films du parc : paquets fermes HI_1_11_0
// 99,498 % -> 99,699 % (38 films), HI_1_12_0 99,917 % -> 99,942 % (96), HI_1_13_0 99,955 % -> 99,985 % (53), les sept pertes etant des
// fermetures fausses (debut de vue B different de celui du decodeur tiers, de 43 bits sur HI_1_12_0 et de plus de 3 600 sur HI_1_11_0) ; HI_1_8_0 et HI_1_4_1 inchanges ; HI_1_10_0
// 96,084 % -> 96,081 % et HI_1_9_0 98,809 % -> 98,803 % : vingt paquets (19 et 1) que la lecture des charges mesurees fermait par la
// fin de leur vue A et que la vue A arretee ne ferme pas, comme v1.0.1. Les cinq films dont v1.0.1 lisait des paquets que v1.2.4 ne lisait
// plus n en perdent plus aucun (35 -> 0), et les sept fermetures fausses de v1.0.1 ne reviennent pas. Les rejeux et les faits cuits
// avant se refusent sur cette revision : la recuisson est une decision de l utilisateur.

// ENTREE `grammar-2026-11-13` (2026-11-13, les passes bit a bit ne tournent plus que derriere la marche des trames). LE DECODAGE
// CHANGE pour deux lectures, et pour elles seules. Jusqu ici, sept passes cherchaient leurs records a chaque bit de chaque paquet
// delta, que la marche l ait lu ou non : creations et pistes des objets du monde, ancrage bipede, visee seule, positions de vehicule,
// creations de bipede, lancers de grenade. Chacune prend desormais les records que la marche a lus, a leur en-tete, par son lecteur
// de toujours, et ne cherche plus que la ou la fermeture de la trame ne prouve pas, pour un slot que la marche n y a pas lu
// ([rendParLAncrage], [finSelonLesTrames]) ; un film sans registre garde la passe entiere. En cuisson, la marche ferme presque toutes
// les trames (26 438 sur 26 439 et, au pire de dix temoins, 40 652 sur 41 019). CE QUI CHANGE : la VISEE SEULE ([ScanBipedAimOnly]) ne
// rend plus une lecture de la passe hors de tout record d une trame fermee — 31 a 281 lectures de moins par film sur dix films de
// l equivalence, toutes sur un slot que la marche n a pas lu dans la trame, la moitie a plus de 90 degres d elevation ; les POSITIONS
// DE VEHICULE ([ScanVehiclePositions]) gagnent les records que la marche lit et que l ancrage manquait, et perdent ses lectures de
// hasard — meme nombre de vehicules sur dix films, 1 a 302 echantillons de plus par film a 1 cm de mediane du trajet deja connu
// (42 cm au plus), 0 ou 1 de moins, echantillons hors de l emprise de la carte 9 -> 0 et 3 -> 0. CE QUI NE CHANGE PAS : creations et
// pistes du monde, ancrage et positions bipedes, creations de bipede, lancers de grenade, motifs de xuid ([source.ChercherMotifs64]),
// table d entites d une image-cle lue une fois par contexte — documents identiques a l octet sur dix temoins et, hors vehicules et
// visee, sur les vingt films de l equivalence. Le canal des lectures bipedes ne juge plus un record pendant la marche : il le porte a
// un journal rejoue a la cloture sous les generations vivantes du film, memes gardes dans le meme ordre (`journal_des_bipedes.go`).
// Les compteurs des passes changent de sens : une ancre de creation de bipede est un record lu, plus un en-tete de hasard (879 ancres
// pour 105 creations -> 105). Duree du decodage seul sur dix temoins : mediane 7,4 s -> 2,5 s, pire 31,8 s -> 7,5 s. Les rejeux et
// les faits cuits avant se refusent sur cette revision : la recuisson est une decision de l utilisateur.

// ENTREE `grammar-2026-11-14` (2026-11-14, les medailles du morceau des temps forts sont rendues avec le fil des morts). LE DECODAGE
// GAGNE UNE LECTURE, et aucune n en change. Le morceau des temps forts etait parse pour ses seules morts ; il l est desormais UNE fois
// pour ses morts ET ses medailles ([scanTempsForts], [medaillesDesTempsForts]) : xuid, instant, poids de tri et indice de medaille
// de chaque evenement `medal`, dans l ordre total instant, xuid, indice ([LecturesDuPont.Medailles], [LecturesDuPont.MedaillesLues]).
// Le fil des morts, son erreur et le verdict du repli du numero sont rendus a l identique ; aucun bit n est lu autrement. CE QUI NE
// CHANGE PAS : le parseur ([ParseHighlightEvents]), le choix du morceau, les morts. Le zero de l horloge des temps forts se mesure a part
// (publication) : sur les films que libfilmshell lit, son zero calcule par les kills tombe a moins de 11 ms du premier paquet du film.
// Les rejeux et les faits cuits avant se refusent sur cette revision : la recuisson est une decision de l utilisateur.

// ENTREE `grammar-2026-11-14.2` (2026-11-14, le statborg se lit jusqu'au dernier siege du film). LA SORTIE DE LA LECTURE DU STATBORG
// CHANGE (`signaux`, que seuls les objectifs consomment ; `killsource` ne l'importe pas). Le balayage des enregistrements d'entite ne
// s'arrete plus au slot 24 : il va jusqu'au dernier siege que les tables de datums du film declarent ([signaux.LaBandeDuStatborg] : index
// contigus a partir de 3 qui levent le composant 57), au plus le slot 100 ; une table absente ou refusee ramene aux huit premiers
// joueurs et se compte (`repli_bande_statborg_par_defaut`). Le lecteur du bloc de datums sort de ce paquet vers `grammar/datums` sans
// changer de lecture. Sur 1 612 films du parc, la derniere ligne lue egale la bande des datums sur 1 608 (les quatre autres ont moins de
// huit joueurs) ; 158 films vont au-dela du slot 24, jusqu'au slot 62. Les rejeux et les faits cuits avant se refusent sur cette
// revision : la recuisson est une decision de l utilisateur.
// ENTREE `grammar-2026-11-14.3` (2026-11-14, qui est present et qui commande quel corps : deux lectures neuves). AUCUN BIT DEJA LU NE
// CHANGE D INTERPRETATION ; deux lectures qui existaient sans appelant de production entrent dans les faits. (1) LES PAQUETS DE TYPE
// 8 ([DecodeRoster]) : [ScanRosterListings] lit les paquets de population de la session que la source a deja decoupes, saute un
// paquet identique au precedent, et range par humain des plages (un xuid sous un siege moteur, du premier paquet qui le nomme au
// premier paquet qui ne le nomme plus) : arrivee et depart dates au paquet pres. Aucun balayage de bit hors de la grammaire du
// paquet ; une lecture qui deborde ou n atteint pas le nombre d entrees annonce est ecartee et comptee. Mesure sur les 1 455 films
// des documents du parc : 1 450 lus (cinq sans table des joueurs lisible), 49 625 paquets de type 8, 7 641 distincts, 0 refuse ;
// aucune des 14 345 entrees humaines du roster publie n est sans plage. (2) LE JOUEUR AUX COMMANDES : la tete d `i18 unit-control`
// ([consumeUnitControl]), que le deserialiseur publiait sur [EtatControleUnite] sans observateur de production, est recue par le
// canal des etats de mouvement sur le meme crochet, y compris sur un film qui ne transmet aucun etat. Le record NEW d un bipede
// porte la tete ; son slot n est lie qu apres la traversee : la lecture attend la fin de la trame ([controleDesUnites]), puis est
// gardee si la table d entites de la marche lie le slot au bipede, comptee sinon. Le canal des etats de mouvement perd le filtre
// « sur un film qui les transmet » pour ce seul composant. Les rejeux et les faits cuits avant se refusent sur cette revision :
// la recuisson est une decision de l utilisateur.

// ENTREE `grammar-2026-11-14.4` (2026-11-14, horloge de manche, comptes a rebours de reapparition, regles de la variante).
// LA MARCHE DES TRAMES RENDUE DE NOUVELLES LECTURES ; AUCUN BIT DEJA LU NE CHANGE D INTERPRETATION. Un canal de la marche
// ([canalDesHorloges]) range, dans l ordre du flux, les lectures que la couche de capture rend de deux composants qu elle
// traversait sans les garder : l horloge de manche de l entite moteur (`game-engine-round-timer-component`, ti=0 i5 : limite, temps de la
// minuterie, etat sur 5 bits) et le compte a rebours de reapparition de l entite joueur du moteur (`player-respawn-timer-component`,
// ti=5 i1).
// L entite joueur du moteur est nommee par l index de joueur que son etat de creation ecrit ([readPlayerWaypointDefaultState]),
// lu a la creation dans une trame delta ([EntityTrace.JoueurDuMoteur]) ou, pour une entite deja la au debut du film, dans l etat
// complet d une image-cle. La lecture de l identite rend en plus la penalite de reapparition apres un suicide, apres une trahison et le
// drapeau des tirs allies de la variante ([lireLesInfosDeMatch]). Les faits cuits avant se refusent sur cette revision et sur
// `replay.SchemaDesFaits` 25 : la recuisson est une decision de l utilisateur.
// ENTREE `grammar-2026-11-15` (2026-11-15, tous les messages de jeu de la vue A, et les scripts du film). LA LECTURE GAGNE DEUX FAITS, aucune
// ne change : la vue A range desormais TOUT message `PlayerGameEventSmall` (genre 82), avec ou sans couple de participants
// ([lecture.VueA.Jeu], [lecture.EvenementDeJeu] : rang R(32), sequence R(8), mot texte du sous-sac, deux premiers participants, masque des
// destinataires) ; un canal de la marche des trames les rend sans les copies renvoyees (meme rang, sequence, mot texte et masque dans la
// seconde, [StatsEvenementsDeJeu]) ; et [ScanScriptsDuFilm] lit les noms de fichier de script du chunk 0. Le rang ne se lit qu'avec le groupe de
// scripts du film : le message ne nomme aucune reponse. CE QUI NE CHANGE PAS : le fil des couples ([lecture.VueA.Fil]), les kills, les
// charges lues ; la lecture de la vue A s'arrete aux memes messages. Les rejeux et les faits cuits avant se refusent sur cette revision :
// la recuisson est une decision de l utilisateur.

// ENTREE `grammar-2026-11-15.2` (2026-11-15, de qui est chaque ligne du tableau des scores). LE DECODAGE GAGNE UNE LECTURE, et aucune n'en
// change. Les images-cles portent, pour chaque entite du tableau des scores, son entree (composant 57 de l'archetype 6 : l'identifiant
// moteur du joueur ou le numero de l'equipe, genre sur huit bits) : [ScanStatborgEntries], un canal de la phase des images-cles qui
// lit les quarante bits du composant que la marche a traverse. Un genre inconnu est une entite vacante, comptee. Sur 000d5950, huit
// lignes de joueur (identifiants 0 a 7, ceux des plages de type 8) et deux d'equipe. Les rejeux et les faits cuits avant se refusent sur
// cette revision : la recuisson est une decision de l'utilisateur.

// ENTREE `grammar-2026-11-15.3` (2026-11-15, la suppression d un bipede se lit). LA MARCHE DES TRAMES RENDUE UNE LECTURE DE PLUS ; AUCUN BIT
// DEJA LU NE CHANGE D INTERPRETATION. Un record DEL retire un slot de la table d entites sans dire l archetype qu il retirait : le canal des
// etats de mouvement tient, depuis les records que la marche range, les slots qu elle a vus porter un bipede avec leur generation (les deux
// bits de tete de l identifiant) et publie le DEL d un tel slot de la meme generation ([suppressionsDeBipedes], [types.BipedDeletion]). Un
// DEL d une marche partie d un debut non prouve ne delie pas le monde : il ne se publie pas, il se compte
// ([types.BipedDeletionStats.DeletionsUnproven]). Aucun balayage de bit hors de la marche. Methode de libfilmshell (Dend, MIT, `src/motion.rs`,
// `src/lives.rs`) : le corps que le jeu supprime encore commande n est pas celui d une mort. Les rejeux et les faits cuits avant se refusent sur
// cette revision et sur `replay.SchemaDesFaits` 28 : la recuisson est une decision de l utilisateur.
// ENTREE `grammar-2026-11-15.4` (2026-11-15, trois apports en une revision : le siege d un occupant de la table du film lu aux listes, les valeurs des proprietes
// reseau de ti=13 lues par la marche, les messages de degat et la vie des bipedes). Les trois lots ont ete integres sur une seule branche ; leurs entrees sont
// gardees ci-dessous avec leurs mesures, prises avant l integration. Les faits cuits avant se refusent sur cette revision : la recuisson est une decision de l utilisateur.
//
// PREMIER APPORT : LE SIEGE D UN OCCUPANT DE LA TABLE DU FILM LU AUX LISTES DE LA SESSION :
// LA TABLE DE `chunk_00` DIT QUI EST
// ASSIS AU COUP D ENVOI ; SON RANG NE DIT L INDEX DE PARTICIPANT QUE DANS LA PLUPART DES FILMS. Sur 31 des 1 606 films dont la table se lit, un a six
// rangs sont listes par les paquets de type 8 sous un autre siege ; sur les 28 de ces films que le decodage complet a juges, la lecture des chunks
// de replication nomme le siege liste sur 46 des 48 lignes (les deux autres n ont aucune lecture), jamais le rang, et le tableau des equipes n a aucun
// joueur au rang sur 33 des 46. Employee telle quelle, la table donnait a l occupant un index que personne ne porte : il perdait son equipe. [SiegesParLesListages]
// remplace le rang par le siege liste quand le premier paquet de type 8 liste tous les occupants de la table sous des sieges deux a deux distincts, et
// la table n a plus de vacant intercale a departager ; tout lecteur du film la lit par [ScanFilmPlayerTableAuxListages] (la cuisson, les bots non epingles, le kill-feed), garde par `TestAucunLecteurNeLitLaTableSansLesListages`. Sans listes lues, ou sans cette unanimite, la table est rendue telle quelle. MESURE AVANT / APRES
// sur les 28 films concernes dont les chunks et les faits sont au parc (document cuit par le decodeur seul, 486 entrees de roster) : 44 entrees changent
// de siege, 25 entrees de roster et 38 pistes passent de "sans equipe" a l equipe de leur index, aucune autre equipe ni aucun autre nom ne change ;
// un seul film (`23ffd885`, vacant intercale refuse jusque la) gagne en plus une vie nommee, l humain parti qu une piste sans nom portait, et une entree
// de roster. Les rejeux et les faits cuits avant se refusent sur cette revision : la recuisson est une decision de l utilisateur.

//
// DEUXIEME APPORT : LES VALEURS DES PROPRIETES RESEAU DE ti=13 LUES PAR LA MARCHE :
// LA MARCHE DES
// TRAMES RENDUE DE NOUVELLES LECTURES ; AUCUN BIT DEJA LU NE CHANGE D INTERPRETATION. La couche de capture rend le variant de chacun des deux
// composants de valeur de ti=13 (`managed-object-property-component` i1, `managed-object-player-masked-property-component` i2..i33 :
// [ManagedPropertyValue]), et un canal de la marche ([canalDesProprietesGereesTrames]) les range, dans l ordre du flux, avec le slot de
// l objet, l horloge du paquet, le champ et l index de joueur du film ([ProprietesGereesLues]). La passe qui ancre les records de ti=13 sur la
// bande de slots ([ScanManagedProperties]) reste, pour les zones : elle retrouve des changements d etat de zone que la liste des entites de
// la trame ne porte pas. Les rejeux et les faits cuits avant se refusent sur cette revision : la jauge de retour du drapeau d un film
// se lit aussi dans la marche (`replay.SchemaDesFaits` 29), et la recuisson est une decision de l utilisateur.
//
// TROISIEME APPORT, PREMIER TEMPS : LES MESSAGES damage_aftermath DE LA VUE A :
// LA LECTURE GAGNE UN FAIT, aucune ne change : la vue A
// range desormais chaque message `damage_aftermath` (genre 0, [chargeDegatsApres]) avec l effet de degat (tag `jpt!`, R(32)), l index de joueur
// de l auteur (R(5), porte a polarite inverse de celle de l effet), l echelle (le premier des trois petits champs) et les deux references de
// l en-tete, l unite blessee et celle qui a cause le degat ([lecture.VueA.Degats], [lecture.DegatDeJoueur]) ; un canal de la marche des trames
// les rend avec l horodatage de la trame ([DegatLu], [StatsDegats]). CE QUI NE CHANGE PAS : les bits consommes par la charge (ils sont les memes),
// le fil des couples, les kills, les charges lues. Les rejeux et les faits cuits avant se refusent sur cette revision et sur
// `replay.SchemaDesFaits` 27 : la recuisson est une decision de l utilisateur.
//
// TROISIEME APPORT, SECOND TEMPS : LA VIE DES BIPEDES LUE POUR LES DEGATS EN POINTS :
// LA LECTURE GAGNE UN CANAL, aucun bit n est lu autrement :
// la marche des trames retient desormais les deux composants de vitalite du bipede (`object-body-vitality`, la sante de -1 a +1, et
// `object-shield-vitality`, le bouclier de 0 a 4, 1 etant plein), que le jeu n ecrit que quand l un d eux change, et la creation de chaque
// bipede ([canalDesVitalites], [VitalitesLues]) ; le message de degat porte en plus la generation de l unite blessee ([DegatLu.GenVictime]). Ces
// lectures servent a relire ce que chaque coup retire a l unite blessee ; aucune regeneration par delai n est supposee. CE QUI NE CHANGE PAS : les
// bits consommes, le fil des couples, les kills, les charges lues. Les rejeux et les faits cuits avant se refusent sur cette revision et sur
// `replay.SchemaDesFaits` 28 : la recuisson est une decision de l utilisateur.
//
// TROISIEME APPORT, REVUE : LA VIE DES BIPEDES RELUE SUR LES LISTES QUE LA MARCHE NE LOCALISE PAS :
// LA LECTURE GAGNE DES TRAMES, aucun bit
// n est lu autrement : le canal des vitalites relit la liste d une trame annoncee que la marche n a pas localisee comme le fait le canal des morts
// ([MarcheDistribuee.recupererLaListe], sans observation, monde intact), et compte les trames qui restent illisibles sans que la vue A s y arrete
// ([StatsVitalites.TramesNonLues], publie par [StatsDegats.VitalesNonLues]). CE QUI NE CHANGE PAS : les bits consommes, le fil des couples, les kills, les
// charges lues, les trames deja localisees. Les rejeux et les faits cuits avant se refusent sur cette revision et sur `replay.SchemaDesFaits` 29.
// REVUE, SECONDE RONDE : le canal de la vie des bipedes compte les listes qu il recupere a largeur libre (`StatsVitalites.LocalisationsALargeurLibre`), comme le canal des
// morts, et les verse au meme repli `repli_localisation_largeur_libre` du rapport du contexte ([MarcheDesTrames.verserLesReplisDesVitalites]). Aucun bit n est lu
// autrement : seul le compte du repli change.

// ENTREE `grammar-2026-11-22` (2026-11-22, un enregistrement du tableau dont un compteur de la manche ne se decode pas est abandonne en entier ; les valeurs figees des manches terminees se lisent). LA LECTURE
// DU TABLEAU DES SCORES CHANGE : le balayage ancrait de faux enregistrements du tableau dans une liste repetitive d une autre structure ; leur lecture s arretait en
// route apres un premier composant de valeurs (corps a corps 0, grenades 16), et ce faux pouvait gagner la selection de la suite. Les composants 0 a 27 sont relus en
// entier par toute emission du moteur : un arret sur l un d eux prouve un ancrage fortuit, et l enregistrement n est plus rendu ([signaux], `statborg.go`). Un arret sur
// un composant d un autre lecteur (valeurs figees, issues, entree) garde ce qui precede. Sortie changee : `scoreTimeline` (compteurs nommes, assistances, quelques points
// de courbe faux) et `roundsUnplayed` sur certains films.
// LES VALEURS FIGEES SE LISENT (meme valeur de revision, la version n etant pas publiee) : le composant `28 + k` porte, pour chaque manche terminee, la paire de valeurs du composant `k` (un masque de 32 bits des manches
// terminees, puis par bit pose un bit de defaut et une valeur de largeur variable, deux fois). Le lecteur des enregistrements ([signaux], `statborg_figees.go`) les range par composant dans `types.StatRecord.Frozen`
// (type `types.StatFrozen`) ; un composant fige illisible, une valeur hors du domaine ou un bit de masque au-dela de la manche 7 abandonne l enregistrement en entier, comme un compteur, et le compte ; un enregistrement
// sans compteur de la manche n a pas de numero de manche et n est pas rendu. Les composants 56 et 57 ne sont toujours pas lus par ce lecteur. Sortie changee en plus : les valeurs figees de chaque enregistrement, les
// compteurs de replis du tableau (`repli_enregistrement_statborg_abandonne`, `repli_composants_statborg_arretes`), et quelques entrees fortuites de `roundsUnplayed` en moins. Les compteurs de la manche, leur ordre et
// leur largeur ne changent pas. Les rejeux et les faits cuits avant se refusent sur cette revision (`replay.SchemaVersion` 126) : la recuisson est
// une decision de l utilisateur.

// ENTREE `grammar-2026-11-22.2` (2026-11-22, les rechargements et la commande de deplacement tenue se lisent ; les tirs se lisent a tout rang d une trame ; le parent d une piece montee se lit a sa creation). LA LECTURE GAGNE DEUX
// CANAUX ET EN CHANGE DEUX, et ces trois apports se rangent sous une seule revision : aucun bit deja lu ne change d interpretation, mais la marche rend des faits qu elle ne rendait pas, et les tirs qu elle rendait augmentent.
// Les rejeux et les faits cuits avant se refusent sur cette revision et sur `replay.SchemaDesFaits` 30 : la recuisson est une decision de l utilisateur.
//
// PREMIER APPORT : LES RECHARGEMENTS D ARME (genre 38 de la vue A). Un message `weapon_reload` porte trois references d en-tete (un `R(1)` de garde puis une reference de domaine chacune), quatre booleens `R(1)` et
// l index de joueur de l auteur (`R(1)`, puis `R(5)` s il est a zero) : [chargeRechargement]. Seule la premiere reference est presente sur les films mesures, c est l unite qui recharge ; aucun champ ne nomme l arme, et le sens des
// quatre booleens n est pas etabli : ils sont lus et ne sont pas publies. Un canal de la marche ([canalDesRechargements]) rend un rechargement par (trame, auteur, references) : le jeu ecrit le meme message deux fois dans la meme
// trame pour environ trois sur dix, et ces copies se comptent ([types.ReloadStats.Copies]) sans devenir des rechargements ; deux messages d une trame qui ne different que par leurs booleens sont un seul rechargement. Les trames
// dont la vue A s arrete sur un message non lu perdent les messages qui les suivent : elles se comptent ([types.ReloadStats.Stopped]), et le compte d un film qui en porte est un plancher. Etape observee `reloads` (et `reloads.stats`),
// calque `reloads` de `layers` (garde : le canal a lu au moins une trame).
//
// SECOND APPORT : LA COMMANDE DE DEPLACEMENT TENUE. L entree de controle d un participant (genre 0, bloc de 0x68 octets) porte, hors de son bloc d action, le couple analogique du deplacement (`R(6) R(6)`, le code `0x1f` est le zero
// exact) et un champ facultatif de cinq bits que ni l un ni l autre ne publie. Le meme collecteur que le tir continu ([collecteurTirContinu], [commandes_de_controle.go]) plie l etat de la commande comme celui d une gachette : tenue de la
// premiere entree lue dont un scalaire n est pas nul a la premiere dont les deux le sont, un paquet dont la vue de controle n est pas lue etant un trou que l on porte sans le combler. Etapes observees `controlHolds` et
// `controlHolds.stats`. Le champ de cinq bits est lu, son bit de poids 4 ressemble au bouton de saut, mais il ne s accorde qu a 63 % avec le saut derive de la vitesse verticale (le seuil de publication est 90 %) : il n est pas publie.
// L accroupissement (`i29`) et la glissade (`i62`) ne se lisent que sur les films dont la version les annonce : aucun film de version 37 et plus ne les annonce, et leur absence n est pas une mesure de ce que les joueurs ont fait.
//
// TROISIEME APPORT : LES TIRS A TOUT RANG D UNE TRAME. Un evenement de tir etait le seul record de tir en tete de la liste d une trame delta ; c est desormais tout message de tir (genre 36) de la vue A, a tout rang
// ([canalDesTirs.tirsApresLaTete], [lecture.MessageDeTir]) : tireur, numero de tir, drapeaux, unite et arme du message. Un tir de rang 1 et plus n a pas de record modal de visee : `HasAim` y reste faux. L etape `fire` rend plus d evenements.
//
// QUATRIEME APPORT : LE PARENT D UNE PIECE MONTEE SE LIT A SA CREATION. Le record de creation d une piece montee (tourelle du Warthog, du Falcon, du Wraith, du Scorpion) annonce `i10` (`object-parent-state`) quand le film rattache la
// piece a son chassis, et son handle (categorie 1 de `FUN_1406d3140`, base `0x200`, les deux bits de queue etant la generation) designe la vie du chassis, y compris quand le chassis est cree apres la piece (225 ms mesures).
// [readCreationParent] relit le record par la boucle de composants de production avec son propre curseur : aucun bit du chemin de creation ne change. `types.EquipmentCreation` gagne `HasParent` et `Parent` ; une lecture absente
// n est pas une lecture « sans parent ». Le plus ancien format (31) ne rattache pas ses pieces par `i10`.
//
// CE QUI MONTE AVEC ELLE : `replay.SchemaDesFaits` 29 -> 30 (le complement de la section 1 porte les deux canaux, la creation d equipement porte son parent), `grammar.Rev` seule des revisions de couche. `source.Rev`, `profile.Rev`,
// `killsource.Rev` et `objectives.Rev` gardent leur valeur : les types `ControlHold`, `ReloadRead` et les champs neufs de `EquipmentCreation` entrent dans leur fermeture sans qu aucune lecture d octets, aucune ligne de kill et
// aucun objectif ne change ; leurs goldens sont refiges a revision constante.
//
// LA MESURE (le corpus d equivalence, 20 films, un passage par fusion) : l etape `reloads` rend 9 055 lectures sur les 20 films (12 708 messages lus, 3 653 copies, 9 026 rechargements publies sur 16 films ; les 4 films dont la vue A
// s arrete tot n en lisent aucun) ; l etape `controlHolds` rend 27 325 commandes tenues ; l etape `fire` passe de 44 837 a 59 249 evenements de tir sur 16 films, les 4 autres (builds dont la vue A s arrete tot) ne changent pas ;
// la lecture du parent ne change aucune etape de la marche : les etapes `pads` (20 films) et `vehicles` (9 films) changent de digest parce que la creation d equipement porte deux champs de plus, dont, pour les 142 pieces montees de 9 films, le parent lu.

// ENTREE `grammar-2026-11-23` (2026-11-23, les charges des genres 27, 117 et 93 inhibe se lisent, et celle du genre 13 sur HI_1_11_0 seul ; une trame renvoyee en entier n est plus comptee deux fois ; les repliques, les annonces et les gestes de la vue A se lisent ; l accroupi, le saut et le pas des entrees de controle se lisent). LA LECTURE GAGNE
// TROIS CANAUX ET QUATRE CHARGES, et ces apports se rangent sous une seule revision : aucun bit deja lu ne change de largeur, mais des messages qui arretaient la lecture d une trame se lisent, la marche rend des faits qu elle ne rendait pas, et un film en rend d autres comptes.
// Les rejeux et les faits cuits avant se refusent sur cette revision, sur `replay.SchemaVersion` 127 et sur `replay.SchemaDesFaits` 31 : la recuisson est une decision de l utilisateur.
//
// PREMIER APPORT : QUATRE CHARGES COURTES DE LA VUE A. `motor_system_interruption` (13), `controlToggleResponse` (27) et `EquipmentTranslocatorTeleportEffects` (117) n avaient pas de lecteur de charge : le premier message de l un de ces genres arretait la lecture de la vue A d une trame, et les messages
// suivants etaient perdus. Les largeurs sont celles de la bibliotheque libfilmshell de Den (OpenSpartan, MIT), temoin de la lecture, prouvees par le film sur les seuls builds ou elles sont ouvertes (voir l entree `grammar-2026-11-24`). `activate_spartan_ability` (93) ne se lit que dans son cas inhibe, le seul que les films montrent ; tout autre
// cas continue d arreter la lecture. Les valeurs lues se traversent : aucune n est publiee (`vue_a_charges_effets.go`).
//
// SECOND APPORT : UNE TRAME RENVOYEE EN ENTIER N EST PLUS COMPTEE DEUX FOIS. Une trame d au moins deux messages de jeu dont tous repetent, a cle identique, ceux d une meme trame de moins de trois secondes est une trame que le jeu renvoie : elle est ecartee et ses messages comptent comme copies ; la fenetre
// d une seconde reste pour un message isole (`evenements_de_jeu.go`).
//
// TROISIEME APPORT : LES REPLIQUES, LES ANNONCES ET LES GESTES. Un canal de la marche rend les messages `AIDialog` (75) et `Dialogue2D` (76) ([canalDesRepliques]) ; la lecture des deux genres existait, le canal les range. Un autre canal rend les gestes : `biped_throw_initiate` (39, `FUN_140c6a58c`) et
// `biped_melee_initiate` (40, `FUN_140ff8d70`) finissaient sur l index de joueur de l auteur (`FUN_1407f2058` : `R(1)`, puis `R(5)` s il est a zero) que leur charge jetait ; elle le range sur le lecteur et la lecture recueille le message ([recueillirUnGeste], [lecture.MessageDeGeste], seize octets). Le lecteur de
// l index de joueur est partage ([lireJoueurDuMessage]) avec la charge du rechargement. Le canal des rechargements rend aussi, pour chaque rechargement, le nombre de copies de son message ([types.ReloadRead.Copies]). Etapes observees `repliques`, `repliques.stats`, `gestures` et `gestures.stats`.
//
// QUATRIEME APPORT : LES COMMANDES D ACCROUPI ET DE SAUT, ET LE PAS DES ENTREES. Le collecteur du tir continu plie deux commandes tenues de plus a partir de l entree de controle : le bit de valeur 1 des drapeaux `+0x14` (le bouton d accroupi, aussi celui de la glissade) et le bit de valeur 4 (le bouton de
// saut), par le meme pliage des entrees et des trous que la commande de deplacement (les natures `natureAccroupi` et `natureSaut`). Un champ de drapeaux absent ne tient aucun bouton (`-1 & 4` n est pas nul : l absence se teste avant le masque). Le collecteur mesure aussi le pas des entrees de controle, l ecart le plus
// frequent entre deux entrees lues d une meme place sans trou entre elles ([pasDesEntrees]) : le fait du film qui separe les films enregistres a pleine cadence (16,7 ms) de ceux enregistres a la moitie (33,3 ms), et que ni la build ni la version du film ne separent.
//
// CINQUIEME APPORT : LE JEU D ARMES DESIRE DES TRAMES DELTA ET LE HANDLE DU PROJECTILE D UN LANCER. Un canal de la marche rend le composant `i42` du bipede (l emplacement d arme desire en main principale et en seconde main) a chaque changement, par slot de piste ([canalDuJeuDArmes]) : le crochet `DesiredWeaponSetHook` prend deux
// parametres, et la porte des essais le liste ; il ne lisait que les images-cles. Le record NEW d un lancer de grenade garde le handle du projectile qu il cree (`GrenadeThrow.Life`, `HasLife`), lu a son en-tete ([grenade_events.go]) : il lie ensuite le message `projectile_detonate` au lancer. Aucun bit lu ne change de largeur.
//
// SIXIEME APPORT : LES PROJECTILES COLLES. Un canal de la marche ([canalDesProjectilesColles]) lit les records de projectile (ti=41) dont l etat de parente `i10` ouvre la branche « attache », creation ou delta, et relit l etat de creation de chaque projectile (instant, type : le mot de 32 bits du tag `proj`, createur : la reference de categorie 1), y compris le projectile qui nait colle.
// Les lectures portent le parent, le noeud et la pose brute (trois entiers de 16 bits). Aucun bit deja lu ne change de largeur ; les listes illisibles se recuperent a largeur libre et se comptent.
//
// SEPTIEME APPORT : LA TABLE DES REPONSES DE JEU ENTRE DANS L EMPREINTE. `reponsesdejeu/data/reponses.tsv` nomme et chiffre les actions de score que la couche fait lire (`scoreActions`) mais n etait importee que par la publication : une ligne de plus ne changeait aucune revision. La couche la declare desormais comme paquet associe
// (`revision.Couche.Associes`, `reponsesdejeu`). AUCUNE SORTIE NE CHANGE : l empreinte change, et c est ce qui rend un rejeu `redecoder` quand la table change. Le golden de perimetre de la couche gagne la ligne `paquet`.
//
// HUITIEME APPORT : L APPARENCE DES SPARTANS EN CODES BRUTS. `ScanRosterAppearances` (`roster_apparences.go`) lit, dans les paquets de type 8 (le roster), le bloc de personnalisation de chaque humain, de la largeur du profil du build ; le bloc de la table de `chunk_00` est nul. Elle ne touche a aucune lecture existante : le meme corps d enregistrement ([DecodeRoster]), un passage de plus sur les memes paquets. Les codes sont opaques (paires cle, valeur de 32 bits puis un reste), le dernier bloc non nul de chaque joueur fait foi ; elle est publiee sous `roster[].appearance` et `coverage.appearance`. AUCUNE LECTURE EXISTANTE NE CHANGE : l empreinte change parce que la couche gagne un fichier.
//
// CE QUI MONTE AVEC ELLE : `replay.SchemaVersion` 126 -> 127 et `replay.SchemaDesFaits` 30 -> 31 (le complement de la section 1 porte les repliques, les gestes, les copies de rechargement, le genre des commandes tenues et le pas des entrees), `objectives.Rev` (la meme integration change la sortie des objectifs : voir
// son entree). `source.Rev`, `profile.Rev` et `killsource.Rev` : aucune lecture d octets ne change dans leur couche ; leurs goldens ne se refigent que si un test le demande (`killsource.Rev` hache la valeur de `grammar.Rev`).

// ENTREE `grammar-2026-11-24` (2026-11-24, les charges courtes ne s ouvrent que build par build ; le canal des projectiles colles borne la naissance a la vie). LA LECTURE CHANGE.
//
// (1) LES CHARGES DES GENRES 13, 27, 93 ET 117 SONT OUVERTES BUILD PAR BUILD, derriere le profil (`profile.EcritureDuBuild`, quatre champs neufs) et non plus sur tout build : genre 13 sur HI_1_11_0 ; genre 27 sur HI_1_11_0 et HI_1_13_0 ; genre 93 (charge inhibee seule) sur HI_1_11_0, HI_1_12_0 et HI_1_13_0 ; genre 117 sur HI_1_12_0 et HI_1_13_0. Ailleurs, le premier message du genre arrete de nouveau la lecture de la trame, arret compte (`stoppedFrames`), comme avant la version. Mesure : toutes les occurrences lues sur un build ouvert sont dans une trame fermee au bit, au moins cinq par genre ; HI_1_10_0 et HI_1_9_0 arretent la trame au message suivant dans 22 a 23 % des cas (2,5 % en general). Sur 450 films : HI_1_10_0, arrets au message suivant 8/9/2/9 -> 0/0/0/0 et arrets sur le genre 0 -> 38/33/37/43 ; les autres builds ne changent que d une trame arretee (HI_1_11_0 14 sur le genre 117, abstention).
// (2) LE CANAL DES PROJECTILES COLLES BORNE LA NAISSANCE A LA VIE : une vie finit a la suppression du slot ou a la creation suivante sous ce slot (une generation differente en ouvre une autre, sans naissance) ; une cle reutilisee n herite plus de la naissance d une vie ancienne ; seul le premier collage de chaque vie est rendu, les suivants sont comptes (`Repeats`). Mesure sur 499 films : 407 repetitions retirees sur 48 198 lectures, aucune naissance d une autre vie n a ete trouvee ; age maximal d une naissance publiee 12,8 s -> 6,2 s.
//
// `profile.Rev` et `objectives.Rev` montent derriere cette entree (voir leur propre chronique) ; `killsource.Rev` hache la valeur de `grammar.Rev`.
