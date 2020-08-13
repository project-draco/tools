# Draco refactoring recommendation tool

The Draco approach consists in the following steps:
1. produce a fine-grained change history of the source-code, where each commit refers to methods or
fields instead of files;
2. compute co-change dependencies, which results in
a graph where nodes represent methods or fields and edges represent a
co-change dependency between them (it is used the metrics _support count_ and
_confidence_ to determine a co-change dependency);
3. Optionaly compute co-change clusters, using a multi-objective genetic algorithm
(a co-change cluster is defined as a set of source code entities
that are strongly co-change dependent on each other);
4. compute _evolutionary smells_, that are identified using one of the following options:
    * co-change clusters: when a co-change cluster contains methods or fields from different classes,
        and at least one of them does not have any dependency (static or co-change)
        upon another element from the same class where it is declared;
    * co-change dependencies: if exists a co-change dependency between two methods or fields from different classes,
      and at least one of them does not have any dependency (static or co-change)
        upon another element from the same class where it is declared;
5. recommend move method or field refactorings that remove such smells.

## Producing Fine-grained Change History

In summary, to transform a regular change history into
a fine-grained change history, we analyze each source-code artifact
within a commit to discover which fine-grained elements have been modified.
We take advantage of [Kenja](https://github.com/niyaton/kenja),
a software utility that produces fine-grained change history
from Git repositories.

## Detecting evolutionary smells

### Co-change clusters option

This option defines evolutionary smell as the
situation where a co-change cluster contains fine-grained entities from more
than one class, and at least one of the entities does not have any dependency
(static or co-change) upon another entity from the same class.
Please refer to this [paper](https://mcesar.dev/papers/jss2019.pdf) for more details.

### Co-change dependencies option

This option defins evolutionary smell
based solely on co-change dependencies, i.e., it does not rely on the
computation of co-change clusters in order to detect such smells.
Specifically, it identifies these smells by looking for
co-change dependencies of the form <img src="https://render.githubusercontent.com/render/math?math=f \rightarrow C">,
where <img src="https://render.githubusercontent.com/render/math?math=f"> is a fine-grained element (method or field) of a class
<img src="https://render.githubusercontent.com/render/math?math=C_f">, and
<img src="https://render.githubusercontent.com/render/math?math=f"> is co-change dependent of some element from class
<img src="https://render.githubusercontent.com/render/math?math=C">,
where <img src="https://render.githubusercontent.com/render/math?math=C\neq C_f">;
and no dependencies (static or co-change) of the form
<img src="https://render.githubusercontent.com/render/math?math=f\rightarrow f'"> exists, where
<img src="https://render.githubusercontent.com/render/math?math=f\neq f'"> and
<img src="https://render.githubusercontent.com/render/math?math=f'\in C_f">.

Intuitively, when we found a situation similar to the aforementioned,
we can conjecture that perhaps the fine-grained source-code element
has been declared in the wrong place.

The second option for detecting evolutionary smells allows us to
skip the computation of co-change clusters, which is expensive in terms of
time of processing and usage of hardware resources.
It also detects a larger number of evolutionary smells, which in turn,
produces more refactoring recommendations.
On the other hand, the use of co-change clusters migth produce more semantic recommendations,
since co-change clusters of fine-grained source-code elements have high conceptual cohesion
([Oliveira et. al. 2015](http://www.ppca.unb.br/images/Documentos/15-sbes-marcos-rodrigo-guilherme.pdf)).

## Recommending Refactorings

The Draco tool recommends to move a method or field if
(a) the number of dependencies is _reduced_ after applying
the refactoring; and (b) the source class does not have any subclass.
The first constraint can be relaxed, by allowing
to recommend refactorings if the source and destination classes already have
a static dependency between them. This way, no _new_ static dependency
between classes is introduced,
while at least one co-change dependency is removed.
This option allows to produce more refactoring recommendations.

## Usage

### Co-change clusters option

`$ recommender --dot-file=<co-change clusters file> <static mdg file> <co-change mdg file> /dev/null [<inheritance> <field types>]`

### Co-change dependencies option

`$ recommender <static mdg file> <co-change mdg file> /dev/null [<inheritance> <field types>]`
