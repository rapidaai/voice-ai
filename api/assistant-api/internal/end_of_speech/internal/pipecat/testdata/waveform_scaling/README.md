# Waveform Scaling Reference

These deterministic fixtures pin the waveform arithmetic used by the controlled
Pipecat NumPy 1.26.4 reference. Run `generate.py` with NumPy 1.26.4 to regenerate
the input/output SHA-256 values. Go tests reconstruct inputs and compare every
output bit through the digest; they do not require Python or audio downloads.

The cases cover empty and short arrays, eight-lane and 128-sample pairwise
boundaries, the 8192-sample reduction buffer, uneven chunks, full model windows,
quiet DC audio, cancellation, constant audio, and negative zero. Input arithmetic
and byte order are fixed in the generator and checked separately from outputs.

NumPy 2 changes Python-scalar promotion. These fixtures deliberately pin the
NumPy 1.26 arithmetic rather than claiming bitwise equality across versions.
