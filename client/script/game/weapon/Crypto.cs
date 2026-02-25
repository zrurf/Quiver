using System;
using System.Security.Cryptography;
public static class Crypto
{
    private static readonly byte[] Key = Convert.FromBase64String("f+wpFv6PSgGcpULdDiXM+DYQR2KOXezWzcF7smQJBrI=");

    public static byte[] Encrypt(byte[] data)
    {
        using Aes aes = Aes.Create();
        aes.Key = Key;
        aes.GenerateIV();
        using var encryptor = aes.CreateEncryptor();
        byte[] encrypted = encryptor.TransformFinalBlock(data, 0, data.Length);
        byte[] result = new byte[aes.IV.Length + encrypted.Length];
        aes.IV.CopyTo(result, 0);
        encrypted.CopyTo(result, aes.IV.Length);
        return result;
    }

    public static byte[] Decrypt(byte[] data)
    {
        using Aes aes = Aes.Create();
        aes.Key = Key;
        byte[] iv = new byte[aes.BlockSize / 8];
        Array.Copy(data, iv, iv.Length);
        aes.IV = iv;
        using var decryptor = aes.CreateDecryptor();
        return decryptor.TransformFinalBlock(data, iv.Length, data.Length - iv.Length);
    }
}