import java.net.URI;
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;
import java.util.concurrent.*;

public class LoadTest {

    static final String BASE_URL = "http://localhost";

    public static void main(String[] args) throws Exception {
        int users = 50;

        ExecutorService executor = Executors.newFixedThreadPool(20);
        HttpClient client = HttpClient.newHttpClient();

        for (int i = 0; i < users; i++) {
            final int userId = i;

            executor.submit(() -> {
                try {
                    String json = "{\"host_name\":\"user_" + userId + "\"}";

                    HttpRequest createReq = HttpRequest.newBuilder()
                            .uri(URI.create(BASE_URL + "/api/rooms"))
                            .header("Content-Type", "application/json")
                            .POST(HttpRequest.BodyPublishers.ofString(json))
                            .build();

                    HttpResponse<String> createRes = client.send(createReq, HttpResponse.BodyHandlers.ofString());

                    System.out.println("User " + userId + " create: " + createRes.statusCode());

                } catch (Exception e) {
                    System.out.println("Error user " + userId + ": " + e.getMessage());
                }
            });
        }

        executor.shutdown();
        executor.awaitTermination(1, TimeUnit.MINUTES);
        System.out.println("Test completed");
    }
}