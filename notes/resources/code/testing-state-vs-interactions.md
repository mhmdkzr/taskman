# Testing on the Toilet: Testing State vs. Testing Interactions

*By Andrew Trenk*
*This article was adapted from a [Google Testing on the Toilet](http://googletesting.blogspot.com/2007/01/introducing-testing-on-toilet.html) (TotT) episode. You can download a printer-friendly [version](//goo.gl/8x1Ha) of this TotT episode and post it in your office.*

There are typically two ways a unit test can verify that the code under test is working properly: by testing state or by testing interactions. What’s the difference between these?

**Testing state means you're verifying that the code under test returns the right results.**

    public void testSortNumbers() {
      NumberSorter numberSorter = new NumberSorter(quicksort, bubbleSort);
      // Verify that the returned list is sorted. It doesn't matter which sorting
      // algorithm is used, as long as the right result is returned.
      assertEquals(
          new ArrayList(1, 2, 3),
          numberSorter.sortNumbers(new ArrayList(3, 1, 2)));
    }

**Testing interactions means you're verifying that the code under test calls certain methods properly.**

    public void testSortNumbers_quicksortIsUsed() {
      // Pass in mocks to the class and call the method under test.
      NumberSorter numberSorter = new NumberSorter(mockQuicksort, mockBubbleSort);
      numberSorter.sortNumbers(new ArrayList(3, 1, 2));
      // Verify that numberSorter.sortNumbers() used quicksort. The test should
      // fail if mockQuicksort.sort() is never called or if it's called with the
      // wrong arguments (e.g. if mockBubbleSort is used to sort the numbers).
      verify(mockQuicksort).sort(new ArrayList(3, 1, 2));
    }

The second test may result in good code coverage, but it doesn't tell you whether sorting works properly, only that quicksort.sort() was called. **Just because a test that uses interactions is passing doesn't mean the code is working properly**. This is why **in most cases, you want to test state, not interactions.**
In general, **interactions should be tested when correctness doesn't just depend on what the code's output is, but also how the output is determined**. In the above example, you would only want to test interactions in addition to testing state if it's important that quicksort is used (e.g. the method would run too slowly with a different sorting algorithm), otherwise the test using interactions is unnecessary.

**What are some other examples of cases where you want to test interactions?**
- **The code under test calls a method where differences in the number or order of calls would cause undesired behavior**, such as side effects (e.g. you only want one email to be sent), latency (e.g. you only want a certain number of disk reads to occur) or multithreading issues (e.g. your code will deadlock if it calls some methods in the wrong order). Testing interactions ensures that your tests will fail if these methods aren't called properly.
- **You're testing a UI where the rendering details of the UI are abstracted away from the UI logic** (e.g. using MVC or MVP). In tests for your controller/presenter, you only care that a certain method of the view was called, not what was actually rendered, so you can test interactions with the view. Similarly, when testing the view, you can test interactions with the controller/presenter.
