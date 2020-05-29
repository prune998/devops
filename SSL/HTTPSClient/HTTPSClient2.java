import java.net.URL;
import javax.net.ssl.HostnameVerifier;

import javax.net.ssl.HttpsURLConnection;
import javax.net.ssl.SSLSession;

public class HTTPSClient2 {
    // Disable the hostname verification for demo purpose
    static {
        HttpsURLConnection.setDefaultHostnameVerifier(new HostnameVerifier() {
            @Override
            public boolean verify(String s, SSLSession sslSession) {
                return true;
            }
        });
    }

    public static void main(String[] args){
        // Initialize configuration
        System.setProperty("javax.net.ssl.trustStore", "./trust.jks");
        System.setProperty("javax.net.ssl.trustStorePassword","mypass");
        System.setProperty("javax.net.ssl.trustStoreType", "jks");

        try{
            URL url = new URL("https://www.google.com");
            HttpsURLConnection client = (HttpsURLConnection) url.openConnection();

            System.out.println("RETURN : "+client.getResponseCode());
        } catch (Exception ex){
            ex.printStackTrace();
        }
    }
}