// scripts/core/Compression.cs
using Godot;
using IronCompress;
using System;

public static class Compression
{
    private static readonly Iron Iron = new Iron();

    public static byte[] Compress(byte[] data)
    {
        var result = Iron.Compress(Codec.Zstd, data);
        return result.AsSpan().ToArray();
    }

    public static byte[] Decompress(byte[] data, int decompressedSize)
    {
        var result = Iron.Decompress(Codec.Zstd, data, decompressedSize);
        return result.AsSpan().ToArray();
    }
}