# Project DRACO tools

Tools that collectively allow to produce refactoring recommendations based on co-change dependencies.
The sequence of steps of the DRACO approach are implemented by the following tools:

- **g2h**: converts a GIT repository to a Historage Repository (HR);
- **mining/co-change**: computes a co-change MDG (Module Dependency Graph) from a HR or GIT repository;
- **clustering**: computes clusters from a MDG (outputs a DOT file format);
- **depfind-converter**: converts a XML produced by depfind to a MDG (depfind is a static dependencies collector),
  or computes an inheritance information file;
- **recommender**: computes evolutionary smells and refactoring recommendations from:
  a co-change MDG,
  a static dependencies MDG,
  an inheritance file,
  and optionaly a co-change clusters DOT file;
- **pruning**: filter a co-change MDG based on minimal support count and confidence metrics.
