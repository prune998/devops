public class PrintCiphers {
  public static void main(String[] args) {
      var sslSocketFactory = javax.net.ssl.SSLServerSocketFactory.getDefault();
      System.out.println("SSLServerSocketFactory -> " + sslSocketFactory.getClass().getName());
      try {
          var getSupportedCipherSuitesMethod = sslSocketFactory.getClass().getMethod("getSupportedCipherSuites");
          String[] ciphers = (String[]) getSupportedCipherSuitesMethod.invoke(sslSocketFactory);
          int i=1;
          for (String c : ciphers) {
              System.out.println(i++ + " " + c);
          }
      } catch(Throwable t) {
          t.printStackTrace();
      }
  }
}