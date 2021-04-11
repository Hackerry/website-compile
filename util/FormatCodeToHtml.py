code = \
'''\
import java.util.Arrays;

public class Main {
    public static void main(String[] args) {
        System.out.println("Starting program...");

        // Initialize array 0-9
        int[] numbers = new int[10];
        for(int i = 0; i < numbers.length; i++) numbers[i] = i;

        // Print out square
        Arrays.stream(numbers).forEach(e -> System.out.println(e*e));

        System.out.println("Program exit");
    }
}
'''

code = code.replace('&','&amp;')
code = code.replace('<','&lt;')
code = code.replace('>','&gt;')

result = "<ul class='ccode'>\n<li><pre>"
for i in code:
    if i == '\n':
        result = result + '</pre></li>\n<li><pre>'
    else:
        result = result + i
result = result[0:-10] + '</ul>'
print(result)